package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	nib "github.com/threefoldtech/zosv2/modules/network/ip"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
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
	// TODO validate each resource

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

func (n *networker) Join(member string, id modules.NetID) (name string, err error) {
	// TODO:
	// 1- Make sure this network id is actually deployed
	// 2- Create a new namespace, then create a veth pair inside this namespace
	// 3- Hook one end to the NR bridge
	// 4- Assign IP to the veth endpoint inside the namespace.
	// 5- return the namespace name

	log.Info().Str("network-id", string(id)).Msg("joining network")

	net, err := n.networkOf(id)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't load network with id (%s)", id)
	}

	// 1- Make sure this network is is deployed
	brName, err := n.bridgeOf(net)
	if err != nil {
		return name, errors.Wrapf(err, "failed to get bridge for network: %v", id)
	}

	br, err := bridge.Get(brName)
	if err != nil {
		return name, err
	}

	netspace, err := namespace.Create(member)
	if err != nil {
		return name, err
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
			Str("namespace", name).
			Str("veth", "eth0").
			Msg("Create veth pair in net namespace")
		hostVeth, containerVeth, err := ip.SetupVeth("eth0", 1500, host)
		if err != nil {
			return errors.Wrapf(err, "failed to create veth pair in namespace (%s)", name)
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

		return netlink.RouteAdd(&netlink.Route{Gw: config.Gateway})
	})

	if err != nil {
		return name, err
	}

	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return name, err
	}

	return name, bridge.AttachNic(hostVeth, br)
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

	enc := json.NewEncoder(file)
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

	dec := json.NewDecoder(file)

	var net modules.Network
	return &net, dec.Decode(&net)
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
