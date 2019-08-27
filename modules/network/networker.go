package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/vishvananda/netlink"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	nib "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/versioned"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

const (
	// ZDBIface is the name of the intefface used in the 0-db network namespace
	ZDBIface = "zdb0"
)

type networker struct {
	identity   modules.IdentityManager
	storageDir string
	tnodb      TNoDB
}

// NewNetworker create a new modules.Networker that can be used over zbus
func NewNetworker(identity modules.IdentityManager, tnodb TNoDB, storageDir string) modules.Networker {
	nw := &networker{
		identity:   identity,
		storageDir: storageDir,
		tnodb:      tnodb,
	}

	return nw
}

var _ modules.Networker = (*networker)(nil)

func validateNetwork(n *modules.Network) error {
	if n.NetID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if n.PrefixZero == nil {
		return fmt.Errorf("PrefixZero cannot be empty")
	}

	if len(n.Resources) < 1 {
		return fmt.Errorf("Network needs at least one network ressource")
	}

	for i, r := range n.Resources {
		nibble := nib.NewNibble(r.Prefix, n.AllocationNR)
		if r.Prefix == nil {
			return fmt.Errorf("Prefix for network resource %s is empty", r.NodeID.Identity())
		}

		peer := r.Peers[i]
		expectedPort := nibble.WireguardPort()
		if peer.Connection.Port != 0 && peer.Connection.Port != expectedPort {
			return fmt.Errorf("Wireguard port for peer %s should be %d", r.NodeID.Identity(), expectedPort)
		}
	}

	if n.Exit == nil {
		return fmt.Errorf("Exit point cannot be empty")
	}

	if n.AllocationNR < 0 {
		return fmt.Errorf("AllocationNR cannot be negative")
	}

	return nil
}

func (n networker) namespaceOf(net *modules.Network) (string, error) {
	nibble, err := n.nibble(net)
	if err != nil {
		return "", err
	}
	return nibble.NetworkName(), nil
}

func (n networker) bridgeOf(net *modules.Network) (string, error) {
	nibble, err := n.nibble(net)
	if err != nil {
		return "", err
	}
	return nibble.BridgeName(), nil
}

func (n *networker) Join(member string, id modules.NetID) (join modules.Member, err error) {
	// TODO:
	// 1- Make sure this network id is actually deployed
	// 2- Create a new namespace, then create a veth pair inside this namespace
	// 3- Hook one end to the NR bridge
	// 4- Assign IP to the veth endpoint inside the namespace.
	// 5- return the namespace name

	log.Info().Str("network-id", string(id)).Msg("joining network")

	net, err := n.networkOf(id)
	if err != nil {
		return join, errors.Wrapf(err, "couldn't load network with id (%s)", id)
	}

	// 1- Make sure this network is is deployed
	brName, err := n.bridgeOf(net)
	if err != nil {
		return join, errors.Wrapf(err, "failed to get bridge for network: %v", id)
	}

	br, err := bridge.Get(brName)
	if err != nil {
		return join, err
	}
	join.Namespace = member
	netspace, err := namespace.Create(member)
	if err != nil {
		return join, err
	}

	defer func() {
		if err != nil {
			namespace.Delete(netspace)
		}
	}()

	var hostVethName string
	err = netspace.Do(func(host ns.NetNS) error {
		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}

		log.Info().
			Str("namespace", join.Namespace).
			Str("veth", "eth0").
			Msg("Create veth pair in net namespace")
		hostVeth, containerVeth, err := ip.SetupVeth("eth0", 1500, host)
		if err != nil {
			return errors.Wrapf(err, "failed to create veth pair in namespace (%s)", join.Namespace)
		}

		hostVethName = hostVeth.Name

		eth0, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		config, err := n.allocateIP(member, net)
		if err != nil {
			return err
		}

		if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: &config.Address}); err != nil {
			return err
		}

		join.IP = config.Address.IP
		return netlink.RouteAdd(&netlink.Route{Gw: config.Gateway})
	})

	if err != nil {
		return join, err
	}

	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return join, err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", hostVeth.Attrs().Name), "1"); err != nil {
		return join, errors.Wrapf(err, "failed to disable ip6 on bridge %s", hostVeth.Attrs().Name)
	}

	return join, bridge.AttachNic(hostVeth, br)
}

// ZDBPrepare sends a macvlan interface into the
// network namespace of a ZDB container
func (n networker) ZDBPrepare() (string, error) {

	netNSName, err := ifaceutil.RandomName("zdb-ns-")
	if err != nil {
		return "", err
	}

	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", err
	}
	defer netNs.Close()

	// find which interface to use as master for the macvlan
	pubIface := DefaultBridge
	if namespace.Exists(PublicNamespace) {
		master, err := publicMasterIface()
		if err != nil {
			return "", errors.Wrap(err, "failed to retrieve the master interface name of the public interface")
		}
		pubIface = master
	}

	macVlan, err := macvlan.Create(ZDBIface, pubIface, netNs)
	if err != nil {
		return "", errors.Wrap(err, "failed to create public mac vlan interface")
	}

	// we don't set any route or ip
	if err := macvlan.Install(macVlan, []*net.IPNet{}, []*netlink.Route{}, netNs); err != nil {
		return "", err
	}

	return netNSName, nil
}

// Addrs return the IP addresses of interface
func (n networker) Addrs(iface string, netns string) ([]net.IP, error) {
	var ips []net.IP

	f := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return errors.Wrapf(err, "failed to get interface %s", iface)
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return errors.Wrapf(err, "failed to list addresses of interfaces %s", iface)
		}
		ips = make([]net.IP, len(addrs))
		for i, addr := range addrs {
			ips[i] = addr.IP
		}
		return nil
	}

	if netns != "" {
		netNS, err := namespace.GetByName(netns)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get network namespace %s", netns)
		}
		defer netNS.Close()

		if err := netNS.Do(f); err != nil {
			return nil, err
		}
	} else {
		if err := f(nil); err != nil {
			return nil, err
		}
	}

	return ips, nil
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(network modules.Network) (string, error) {
	var err error

	if err := validateNetwork(&network); err != nil {
		log.Error().Err(err).Msg("network object format invalid")
		return "", err
	}

	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return "", fmt.Errorf("not network resource for this node: %s", n.identity.NodeID())
	}
	exitNetRes, err := exitResource(network.Resources)
	if err != nil {
		return "", err
	}
	nibble := nib.NewNibble(localResource.Prefix, network.AllocationNR)

	wgKey, err := n.extractPrivateKey(localResource)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract private key from network object")
	}

	// the flow is a bit different if the network namespace already exist or not
	// if it already exists, we skip the all network resource creation
	// and only do the wireguard configuration
	// so any new updated wireguard peer will be updated
	creation := !namespace.Exists(nibble.NetworkName())

	if creation {
		log.Info().Msg("create new network resource")
	} else {
		log.Info().Msg("update existing network resource")
	}

	defer func() {
		if err != nil {
			if err := n.DeleteNetResource(network); err != nil {
				log.Error().Err(err).Msg("during during deletion of network resource after failed deployment")
			}
		}
	}()

	if creation {
		log.Info().Msg("create network resource namespace")
		err = createNetworkResource(localResource, &network)
		if err != nil {
			return "", err
		}
	}

	log.Info().Msg("Generate wireguard config for all peers")
	peers, routes, err := genWireguardPeers(localResource, &network)
	if err != nil {
		return "", err
	}

	log.Debug().
		Str("local prefix", localResource.Prefix.String()).
		Str("exit prefix", exitNetRes.Prefix.String()).
		Msg("configure wireguard exit node")

	// if we are not the exit node, then add the default route to the exit node
	if localResource.Prefix.String() != exitNetRes.Prefix.String() {
		log.Info().Msg("Generate wireguard config to the exit node")
		exitPeers, exitRoutes, err := genWireguardExitPeers(localResource, &network)
		if err != nil {
			return "", err
		}
		peers = append(peers, exitPeers...)
		routes = append(routes, exitRoutes...)
	} else if creation { // we are the exit node and this network resource is being creating
		log.Info().Msg("Configure network resource as exit point")
		err := configNetResAsExitPoint(exitNetRes, network.Exit, network.PrefixZero)
		if err != nil {
			return "", err
		}
	}

	log.Info().
		Int("number of peers", len(peers)).
		Msg("configure wg")

	err = configWG(localResource, &network, peers, routes, wgKey)
	if err != nil {
		return "", err
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer, err := versioned.NewWriter(file, modules.NetworkSchemaLatestVersion)
	if err != nil {
		return "", err
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(network); err != nil {
		return "", errors.Wrap(err, "failed to store network object")
	}

	return nibble.NetworkName(), nil
}

func (n *networker) nibble(network *modules.Network) (*nib.Nibble, error) {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return nil, fmt.Errorf("not network resource for this node: %s", n.identity.NodeID())
	}

	return nib.NewNibble(localResource.Prefix, network.AllocationNR), nil
}

func (n *networker) networkOf(id modules.NetID) (*modules.Network, error) {
	path := filepath.Join(n.storageDir, string(id))
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader, err := versioned.NewReader(file)
	if versioned.IsNotVersioned(err) {
		// old data that doesn't have any version information
		if _, err := file.Seek(0, 0); err != nil {
			return nil, err
		}

		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), file)
	} else if err != nil {
		return nil, err
	}

	var net modules.Network
	dec := json.NewDecoder(reader)

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", modules.NetworkSchemaV1))

	if validV1(reader.Version()) {
		if err := dec.Decode(&net); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown network object version (%s)", reader.Version())
	}

	return &net, nil
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) DeleteNetResource(network modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}
	var (
		nibble     = zosip.NewNibble(localResource.Prefix, network.AllocationNR)
		netnsName  = nibble.NetworkName()
		bridgeName = nibble.BridgeName()
	)
	if err := bridge.Delete(bridgeName); err != nil {
		log.Error().
			Err(err).
			Str("bridge", bridgeName).
			Msg("failed to delete network resource bridge")
		return err
	}

	netResNS, err := namespace.GetByName(netnsName)
	if err != nil {
		return err
	}
	// don't explicitly netResNS.Close() the netResNS here, namespace.Delete will take care of it
	if err := namespace.Delete(netResNS); err != nil {
		log.Error().
			Err(err).
			Str("namespace", netnsName).
			Msg("failed to delete network resource namespace")
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) extractPrivateKey(r *modules.NetResource) (wgtypes.Key, error) {
	key := wgtypes.Key{}

	peer, err := getPeer(r.Prefix.String(), r.Peers)
	if err != nil {
		return key, err
	}
	if peer.Connection.PrivateKey == "" {
		return key, fmt.Errorf("wireguard private key is empty")
	}

	// private key is hex encoded in the network object
	sk := ""
	_, err = fmt.Sscanf(peer.Connection.PrivateKey, "%x", &sk)
	if err != nil {
		return key, err
	}

	decoded, err := n.identity.Decrypt([]byte(sk))
	if err != nil {
		return key, err
	}

	return wgtypes.ParseKey(string(decoded))
}

// publicMasterIface return the name of the master interface
// of the public interface
func publicMasterIface() (string, error) {
	netns, err := namespace.GetByName(PublicNamespace)
	if err != nil {
		return "", err
	}
	defer netns.Close()

	var iface string
	if err := netns.Do(func(_ ns.NetNS) error {
		pl, err := netlink.LinkByName(PublicIface)
		if err != nil {
			return err
		}
		index := pl.Attrs().MasterIndex
		if index == 0 {
			return fmt.Errorf("public iface has not master")
		}
		ml, err := netlink.LinkByIndex(index)
		if err != nil {
			return err
		}
		iface = ml.Attrs().Name
		return nil
	}); err != nil {
		return "", err
	}

	return iface, nil
}

// createNetNS create a network namespace and set lo interface up
func createNetNS(name string) (ns.NetNS, error) {

	netNs, err := namespace.Create(name)
	if err != nil {
		return nil, err
	}

	err = netNs.Do(func(_ ns.NetNS) error {
		return ifaceutil.SetLoUp()
	})
	if err != nil {
		namespace.Delete(netNs)
		return nil, err
	}

	return netNs, nil
}
