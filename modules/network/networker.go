package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules/network/gateway"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	nib "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/network/nr"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/versioned"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
)

const (
	// ZDBIface is the name of the interface used in the 0-db network namespace
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
		return fmt.Errorf("Network needs at least one network resource")
	}

	for i, r := range n.Resources {
		nibble, err := nib.NewNibble(r.Prefix, n.AllocationNR)
		if err != nil {
			return errors.Wrap(err, "allocation prefix is not valid")
		}
		if r.Prefix == nil {
			return fmt.Errorf("Prefix for network resource %s is empty", r.NodeID.Identity())
		}

		peer := r.Peers[i]
		expectedPort := nibble.WireguardPort()
		if peer.Connection.Port != 0 && peer.Connection.Port != expectedPort {
			return fmt.Errorf("Wireguard port for peer %s should be %d", r.NodeID.Identity(), expectedPort)
		}

		if peer.Connection.IP != nil && !peer.Connection.IP.IsGlobalUnicast() {
			return fmt.Errorf("Wireguard endpoint for peer %s should be a public IP, not %s", r.NodeID.Identity(), peer.Connection.IP.String())
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

func (n *networker) Join(containerID string, id modules.NetID) (join modules.Member, err error) {
	// TODO:
	// 1- Make sure this network id is actually deployed
	// 2- Create a new namespace, then create a veth pair inside this namespace
	// 3- Hook one end to the NR bridge
	// 4- Assign IP to the veth endpoint inside the namespace.
	// 5- return the namespace name

	log.Info().Str("network-id", string(id)).Msg("joining network")

	network, err := n.networkOf(id)
	if err != nil {
		return join, errors.Wrapf(err, "couldn't load network with id (%s)", id)
	}

	nodeID := n.identity.NodeID().Identity()
	netRes, err := nr.New(nodeID, network, wgtypes.Key{})
	if err != nil {
		return join, errors.Wrap(err, "failed to load network resource")
	}

	return netRes.Join(containerID)
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
	if namespace.Exists(types.PublicNamespace) {
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

	nodeID := n.identity.NodeID().Identity()

	localResource, err := nr.ResourceByNodeID(nodeID, network.Resources)
	if err != nil {
		return "", err
	}

	privateKey, err := n.extractPrivateKey(localResource)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract private key from network object")
	}

	nr, err := nr.New(nodeID, &network, privateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to load network resource")
	}

	cleanup := func() {
		log.Error().Msg("clean up network resource")
		if err := nr.Delete(); err != nil {
			log.Error().Err(err).Msg("error during deletion of network resource after failed deployment")
		}
	}

	// this is ok if pubNS is nil, nr.Create handles it
	pubNS, _ := namespace.GetByName(types.PublicNamespace)

	log.Info().Msg("create network resource namespace")
	if err := nr.Create(pubNS); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to create network resource")
	}

	if exitNodeNr, isExist := nr.IsExit(); isExist {
		gw := gateway.New(network.PrefixZero, int(network.AllocationNR), exitNodeNr)
		if err := gw.Create(); err != nil {
			cleanup()
			return "", errors.Wrap(err, "failed to create gateway")
		}

		if err := gw.AddNetResource(nr); err != nil {
			cleanup()
			return "", errors.Wrap(err, "failed to add network resource to gateway")
		}
	}

	if err := nr.Configure(); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to configure network resource")
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	file, err := os.Create(path)
	if err != nil {
		cleanup()
		return "", err
	}
	defer file.Close()
	writer, err := versioned.NewWriter(file, modules.NetworkSchemaLatestVersion)
	if err != nil {
		cleanup()
		return "", err
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(network); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to store network object")
	}

	return nr.NamespaceName(), nil
}

func (n *networker) nibble(network *modules.Network) (*nib.Nibble, error) {
	localResource, err := nr.ResourceByNodeID(n.identity.NodeID().Identity(), network.Resources)
	if err != nil {
		return nil, err
	}

	nibble, err := nib.NewNibble(localResource.Prefix, network.AllocationNR)
	if err != nil {
		return nil, err
	}
	return nibble, nil
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
	nodeID := n.identity.NodeID().Identity()
	nr, err := nr.New(nodeID, &network, wgtypes.Key{})
	if err != nil {
		return errors.Wrap(err, "failed to load network resource")
	}

	if err := nr.Delete(); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) extractPrivateKey(r *modules.NetResource) (wgtypes.Key, error) {
	//FIXME zaibon: I would like to move this into the nr package,
	// but this method requires the identity module which is only available
	// on the networker object

	key := wgtypes.Key{}

	peer, err := nr.PeerByPrefix(r.Prefix.String(), r.Peers)
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
	netns, err := namespace.GetByName(types.PublicNamespace)
	if err != nil {
		return "", err
	}
	defer netns.Close()

	var iface string
	if err := netns.Do(func(_ ns.NetNS) error {
		pl, err := netlink.LinkByName(types.PublicIface)
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
