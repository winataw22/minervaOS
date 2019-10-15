package network

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/threefoldtech/zos/pkg/network/ndmz"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"

	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/nr"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/set"
	"github.com/threefoldtech/zos/pkg/versioned"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/network/namespace"

	"github.com/threefoldtech/zos/pkg"
)

const (
	// ZDBIface is the name of the interface used in the 0-db network namespace
	ZDBIface = "zdb0"
)

type networker struct {
	identity   pkg.IdentityManager
	storageDir string
	tnodb      TNoDB
	portSet    *set.UintSet
}

// NewNetworker create a new pkg.Networker that can be used over zbus
func NewNetworker(identity pkg.IdentityManager, tnodb TNoDB, storageDir string) pkg.Networker {
	nw := &networker{
		identity:   identity,
		storageDir: storageDir,
		tnodb:      tnodb,
		portSet:    set.NewUint(),
	}

	return nw
}

var _ pkg.Networker = (*networker)(nil)

func validateNetwork(n *pkg.Network) error {
	if n.NetID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if n.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	if n.IPRange.Nil() {
		return fmt.Errorf("network IP range cannot be empty")
	}

	if len(n.NetResources) < 1 {
		return fmt.Errorf("Network needs at least one network resource")
	}

	for _, nr := range n.NetResources {
		if err := validateNR(nr); err != nil {
			return err
		}
	}

	return nil
}
func validateNR(nr pkg.NetResource) error {

	if nr.NodeID == "" {
		return fmt.Errorf("network resource node ID cannot empty")
	}
	if nr.Subnet.IP == nil {
		return fmt.Errorf("network resource subnet cannot empty")
	}

	if nr.WGPrivateKey == "" {
		return fmt.Errorf("network resource wireguard private key cannot empty")
	}

	if nr.WGPublicKey == "" {
		return fmt.Errorf("network resource wireguard public key cannot empty")
	}

	if nr.WGListenPort <= 0 {
		return fmt.Errorf("network resource wireguard listen port cannot empty")
	}

	for _, peer := range nr.Peers {
		if err := validatePeer(peer); err != nil {
			return err
		}
	}

	return nil
}

func validatePeer(p pkg.Peer) error {
	if p.WGPublicKey == "" {
		return fmt.Errorf("peer wireguard public key cannot empty")
	}

	if p.Subnet.Nil() {
		return fmt.Errorf("peer wireguard subnet cannot empty")
	}

	if len(p.AllowedIPs) <= 0 {
		return fmt.Errorf("peer wireguard allowedIPs cannot empty")
	}
	return nil
}

func (n *networker) Join(networkdID pkg.NetID, containerID string, addrs []string) (join pkg.Member, err error) {
	// TODO:
	// 1- Make sure this network id is actually deployed
	// 2- Create a new namespace, then create a veth pair inside this namespace
	// 3- Hook one end to the NR bridge
	// 4- Assign IP to the veth endpoint inside the namespace.
	// 5- return the namespace name

	log.Info().Str("network-id", string(networkdID)).Msg("joining network")

	network, err := n.networkOf(string(networkdID))
	if err != nil {
		return join, errors.Wrapf(err, "couldn't load network with id (%s)", networkdID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := ResourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return join, err
	}

	netRes, err := nr.New(networkdID, localNR, &network.IPRange.IPNet)
	if err != nil {
		return join, errors.Wrap(err, "failed to load network resource")
	}

	ips := make([]net.IP, len(addrs))
	for i, addr := range addrs {
		ips[i] = net.ParseIP(addr)
	}

	return netRes.Join(containerID, ips)
}

func (n *networker) Leave(networkdID pkg.NetID, containerID string) error {
	log.Info().Str("network-id", string(networkdID)).Msg("leaving network")

	network, err := n.networkOf(string(networkdID))
	if err != nil {
		return errors.Wrapf(err, "couldn't load network with id (%s)", networkdID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := ResourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return err
	}

	netRes, err := nr.New(networkdID, localNR, &network.IPRange.IPNet)
	if err != nil {
		return errors.Wrap(err, "failed to load network resource")
	}

	return netRes.Leave(containerID)
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

// CreateNR implements pkg.Networker interface
func (n *networker) CreateNR(network pkg.Network) (string, error) {
	var err error
	var nodeID = n.identity.NodeID().Identity()

	if err := validateNetwork(&network); err != nil {
		log.Error().Err(err).Msg("network object format invalid")
		return "", err
	}

	b, err := json.Marshal(network)
	if err != nil {
		panic(err)
	}
	log.Debug().
		Str("network", string(b)).
		Msg("create NR")

	netNR, err := ResourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return "", err
	}

	privateKey, err := n.extractPrivateKey(netNR.WGPrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract private key from network object")
	}

	// check if there is a reserved wireguard port for this NR already
	// or if we need to update it
	storedNet, err := n.networkOf(string(network.NetID))

	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		storedNR, err := ResourceByNodeID(nodeID, storedNet.NetResources)
		if err != nil {
			return "", err
		}
		if netNR.WGListenPort != storedNR.WGListenPort {
			if err := n.releasePort(storedNR.WGListenPort); err != nil {
				return "", err
			}
		}
	} else if os.IsNotExist(err) {
		if err := n.reservePort(netNR.WGListenPort); err != nil {
			return "", err
		}
	}

	netr, err := nr.New(network.NetID, netNR, &network.IPRange.IPNet)
	if err != nil {
		return "", err
	}

	cleanup := func() {
		log.Error().Msg("clean up network resource")
		if err := n.releasePort(netNR.WGListenPort); err != nil {
			log.Error().Err(err).Msg("release wireguard port failed")
		}
		if err := netr.Delete(); err != nil {
			log.Error().Err(err).Msg("error during deletion of network resource after failed deployment")
		}
	}

	// this is ok if pubNS is nil, nr.Create handles it
	pubNS, _ := namespace.GetByName(types.PublicNamespace)

	log.Info().Msg("create network resource namespace")
	if err := netr.Create(pubNS); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to create network resource")
	}

	if err := ndmz.AttachNR(string(network.NetID), netr); err != nil {
		return "", errors.Wrapf(err, "failed to attach network resource to DMZ bridge")
	}

	if err := netr.ConfigureWG(privateKey); err != nil {
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
	writer, err := versioned.NewWriter(file, pkg.NetworkSchemaLatestVersion)
	if err != nil {
		cleanup()
		return "", err
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(&network); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to store network object")
	}

	return netr.Namespace()
}

func (n *networker) networkOf(id string) (*pkg.Network, error) {
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

	var net pkg.Network
	dec := json.NewDecoder(reader)

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", pkg.NetworkSchemaV1))

	if validV1(reader.Version()) {
		if err := dec.Decode(&net); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown network object version (%s)", reader.Version())
	}

	return &net, nil
}

// DeleteNR implements pkg.Networker interface
func (n *networker) DeleteNR(network pkg.Network) error {
	netNR, err := ResourceByNodeID(n.identity.NodeID().Identity(), network.NetResources)
	if err != nil {
		return err
	}

	nr, err := nr.New(network.NetID, netNR, &network.IPRange.IPNet)
	if err != nil {
		return errors.Wrap(err, "failed to load network resource")
	}

	if err := nr.Delete(); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}

	if err := n.releasePort(netNR.WGListenPort); err != nil {
		log.Error().Err(err).Msg("release wireguard port failed")
		// TODO: should we return the error ?
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) extractPrivateKey(hexKey string) (string, error) {
	//FIXME zaibon: I would like to move this into the nr package,
	// but this method requires the identity module which is only available
	// on the networker object
	sk, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", err
	}
	decoded, err := n.identity.Decrypt(sk)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func (n *networker) reservePort(port uint16) error {
	err := n.portSet.Add(uint(port))
	if err != nil {
		return errors.Wrap(err, "wireguard listen port already in use, pick another one")
	}

	if err := n.tnodb.PublishWGPort(n.identity.NodeID(), n.portSet.List()); err != nil {
		n.portSet.Remove(uint(port))
		return errors.Wrap(err, "fail to publish wireguard port to bcdb")
	}

	return nil
}

func (n *networker) releasePort(port uint16) error {
	n.portSet.Remove(uint(port))

	if err := n.tnodb.PublishWGPort(n.identity.NodeID(), n.portSet.List()); err != nil {
		// maybe retry a couple of times ?
		// having bdb and the node out of sync is pretty bad
		return errors.Wrap(err, "fail to publish wireguard port to bcdb")
	}

	return nil
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

// ResourceByNodeID return the net resource associated with a nodeID
func ResourceByNodeID(nodeID string, resources []pkg.NetResource) (*pkg.NetResource, error) {
	for _, resource := range resources {
		if resource.NodeID == nodeID {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("not network resource for this node: %s", nodeID)
}
