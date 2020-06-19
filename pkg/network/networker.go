package network

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/termie/go-shutil"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/tuntap"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"

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
	ZDBIface     = "zdb0"
	wgPortDir    = "wireguard_ports"
	networkDir   = "networks"
	ipamLeaseDir = "ndmz-lease"
	ipamPath     = "/var/cache/modules/networkd/lease"
)

const (
	mib = 1024 * 1024
)

type networker struct {
	identity     pkg.IdentityManager
	networkDir   string
	ipamLeaseDir string
	tnodb        client.Directory
	portSet      *set.UintSet

	ndmz ndmz.DMZ
	ygg  *yggdrasil.Server
}

// NewNetworker create a new pkg.Networker that can be used over zbus
func NewNetworker(identity pkg.IdentityManager, tnodb client.Directory, storageDir string, ndmz ndmz.DMZ, ygg *yggdrasil.Server) (pkg.Networker, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	wgDir := filepath.Join(vd, wgPortDir)
	if err := os.MkdirAll(wgDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create wireguard port cache directory: %w", err)
	}

	nwDir := filepath.Join(vd, networkDir)
	ipamLease := filepath.Join(vd, ipamLeaseDir)

	oldPath := filepath.Join(ipamPath, "ndmz")
	newPath := filepath.Join(ipamLease, "ndmz")
	if _, err := os.Stat(oldPath); err == nil {
		if err := shutil.CopyTree(oldPath, newPath, nil); err != nil {
			return nil, err
		}
		_ = os.RemoveAll(ipamPath)
	}

	//TODO: remove once all the node have move the network into the volatile directory
	if _, err = os.Stat(storageDir); err == nil {
		if err := copyNetworksToVolatile(storageDir, nwDir); err != nil {
			return nil, fmt.Errorf("failed to copy old networks directory: %w", err)
		}
	}

	nw := &networker{
		identity:     identity,
		tnodb:        tnodb,
		networkDir:   nwDir,
		ipamLeaseDir: ipamLease,
		portSet:      set.NewUint(wgDir),

		ygg:  ygg,
		ndmz: ndmz,
	}

	// always add the reserved yggdrasil port to the port set so we make sure they are never
	// picked for wireguard endpoints
	for _, port := range []int{yggdrasil.YggListenTCP, yggdrasil.YggListenTLS, yggdrasil.YggListenLinkLocal} {
		if err := nw.portSet.Add(uint(port)); err != nil && errors.Is(err, set.ErrConflict{}) {
			return nil, err
		}
	}
	if err := nw.publishWGPorts(); err != nil {
		return nil, err
	}

	return nw, nil
}

func copyNetworksToVolatile(src, dst string) error {
	log.Info().Msg("move network cached file to volatile directory")
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	copy := func(src string, dst string) error {
		log.Info().Str("source", src).Str("dst", dst).Msg("copy file")
		// Read all content of src to data
		data, err := ioutil.ReadFile(src)
		if err != nil {
			return err
		}
		// Write data to dst
		return ioutil.WriteFile(dst, data, 0644)
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		s := filepath.Join(src, info.Name())
		d := filepath.Join(dst, info.Name())
		if err := copy(s, d); err != nil {
			return err
		}
	}
	return nil
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
		return fmt.Errorf("network needs at least one network resource")
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

	if nr.WGListenPort == 0 {
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

func (n *networker) Ready() error {
	return nil
}

func (n *networker) Join(networkdID pkg.NetID, containerID string, addrs []string, publicIP6 bool) (join pkg.Member, err error) {
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
	localNR, err := resourceByNodeID(nodeID, network.NetResources)
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

	join, err = netRes.Join(containerID, ips, publicIP6)
	if err != nil {
		return join, errors.Wrap(err, "failed to load network resource")
	}

	if publicIP6 {
		netNs, err := namespace.GetByName(join.Namespace)
		if err != nil {
			return join, errors.Wrap(err, "failed to found a valid network interface to use as parent for 0-db container")
		}
		defer netNs.Close()

		hw := ifaceutil.HardwareAddrFromInputBytes([]byte(containerID))

		if err = n.createMacVlan("pub", hw, nil, nil, netNs); err != nil {
			return join, errors.Wrap(err, "failed to create public macvlan interface")
		}
	}

	return join, nil
}

func (n *networker) Leave(networkdID pkg.NetID, containerID string) error {
	log.Info().Str("network-id", string(networkdID)).Msg("leaving network")

	network, err := n.networkOf(string(networkdID))
	if err != nil {
		return errors.Wrapf(err, "couldn't load network with id (%s)", networkdID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := resourceByNodeID(nodeID, network.NetResources)
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
func (n networker) ZDBPrepare(hw net.HardwareAddr) (string, error) {
	netNSName, err := ifaceutil.RandomName("zdb-ns-")
	if err != nil {
		return "", err
	}

	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", err
	}
	defer netNs.Close()

	var (
		ips    []*net.IPNet
		routes []*netlink.Route
	)

	if n.ygg != nil {
		ip, err := n.ygg.SubnetFor(hw)
		if err != nil {
			return "", err
		}

		ips = []*net.IPNet{
			{
				IP:   ip,
				Mask: net.CIDRMask(64, 128),
			},
		}

		gw, err := n.ygg.Gateway()
		if err != nil {
			return "", err
		}

		routes = []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("200::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gw: gw.IP,
				// LinkIndex:... this is set by macvlan.Install
			},
		}

	}

	return netNSName, n.createMacVlan(ZDBIface, hw, ips, routes, netNs)
}

func (n networker) createMacVlan(iface string, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netNs ns.NetNS) error {
	macVlan, err := macvlan.Create(iface, n.ndmz.IP6PublicIface(), netNs)
	if err != nil {
		return errors.Wrap(err, "failed to create public mac vlan interface")
	}

	log.Debug().Str("HW", hw.String()).Str("macvlan", macVlan.Name).Msg("setting hw address on link")
	// we don't set any route or ip
	if err := macvlan.Install(macVlan, hw, ips, routes, netNs); err != nil {
		return err
	}

	return nil
}

// SetupTap interface in the network resource. We only allow 1 tap interface to be
// set up per NR currently
func (n *networker) SetupTap(networkID pkg.NetID) (string, error) {
	log.Info().Str("network-id", string(networkID)).Msg("Setting up tap interface")

	network, err := n.networkOf(string(networkID))
	if err != nil {
		return "", errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := resourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return "", err
	}

	netRes, err := nr.New(networkID, localNR, &network.IPRange.IPNet)
	if err != nil {
		return "", errors.Wrap(err, "failed to load network resource")
	}

	bridgeName, err := netRes.BridgeName()
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace bridge")
	}

	tapIface, err := tapName(networkID)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	_, err = tuntap.CreateTap(tapIface, bridgeName)

	return tapIface, err
}

// RemoveTap in the network resource.
func (n *networker) RemoveTap(networkID pkg.NetID) error {
	log.Info().Str("network-id", string(networkID)).Msg("Removing tap interface")

	tapIface, err := tapName(networkID)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

// GetSubnet of a local network resource identified by the network ID
func (n networker) GetSubnet(networkID pkg.NetID) (net.IPNet, error) {
	network, err := n.networkOf(string(networkID))
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := resourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return net.IPNet{}, err
	}

	return localNR.Subnet.IPNet, nil
}

// GetDefaultGwIP returns the IP(v4) of the default gateway inside the network
// resource identified by the network ID on the local node
func (n networker) GetDefaultGwIP(networkID pkg.NetID) (net.IP, error) {
	network, err := n.networkOf(string(networkID))
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	nodeID := n.identity.NodeID().Identity()
	localNR, err := resourceByNodeID(nodeID, network.NetResources)
	if err != nil {
		return nil, err
	}

	// only IP4 atm
	ip := localNR.Subnet.IP.To4()
	if ip == nil {
		return nil, errors.New("nr subnet is not valid IPv4")
	}

	// defaut gw is currently implied to be at `x.x.x.1`
	// also a subnet in a NR is assumed to be a /24
	ip[len(ip)-1] = 1

	return ip, nil
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
	defer func() {
		if err := n.publishWGPorts(); err != nil {
			log.Warn().Err(err).Msg("failed to publish wireguard port to BCDB")
		}
	}()

	var err error
	var nodeID = n.identity.NodeID().Identity()

	if err := validateNetwork(&network); err != nil {
		log.Error().Err(err).Msg("network object format invalid")
		return "", err
	}

	log.Info().Str("network", network.Name).Msg("create network resource")

	netNR, err := resourceByNodeID(nodeID, network.NetResources)
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
		storedNR, err := resourceByNodeID(nodeID, storedNet.NetResources)
		if err != nil {
			return "", err
		}
		if err := n.releasePort(storedNR.WGListenPort); err != nil {
			return "", err
		}
	}

	if err := n.reservePort(netNR.WGListenPort); err != nil {
		return "", err
	}

	netr, err := nr.New(network.NetID, netNR, &network.IPRange.IPNet)
	if err != nil {
		return "", err
	}

	cleanup := func() {
		log.Error().Msg("clean up network resource")
		if err := netr.Delete(); err != nil {
			log.Error().Err(err).Msg("error during deletion of network resource after failed deployment")
		}
		if err := n.releasePort(netNR.WGListenPort); err != nil {
			log.Error().Err(err).Msg("release wireguard port failed")
		}
	}

	// this is ok if pubNS is nil, nr.Create handles it
	pubNS, _ := namespace.GetByName(types.PublicNamespace)

	log.Info().Msg("create network resource namespace")
	if err := netr.Create(pubNS); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to create network resource")
	}

	if err := n.ndmz.AttachNR(string(network.NetID), netr, n.ipamLeaseDir); err != nil {
		return "", errors.Wrapf(err, "failed to attach network resource to DMZ bridge")
	}

	if err := netr.ConfigureWG(privateKey); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to configure network resource")
	}

	if err := n.storeNetwork(network); err != nil {
		cleanup()
		return "", errors.Wrap(err, "failed to store network object")
	}

	return netr.Namespace()
}

func (n *networker) storeNetwork(network pkg.Network) error {
	// map the network ID to the network namespace
	path := filepath.Join(n.networkDir, string(network.NetID))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := versioned.NewWriter(file, pkg.NetworkSchemaLatestVersion)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(&network); err != nil {
		return err
	}

	return nil
}

// DeleteNR implements pkg.Networker interface
func (n *networker) DeleteNR(network pkg.Network) error {
	defer func() {
		if err := n.publishWGPorts(); err != nil {
			log.Warn().Msg("failed to publish wireguard port to BCDB")
		}
	}()

	netNR, err := resourceByNodeID(n.identity.NodeID().Identity(), network.NetResources)
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
	path := filepath.Join(n.networkDir, string(network.NetID))
	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) networkOf(id string) (*pkg.Network, error) {
	path := filepath.Join(n.networkDir, string(id))
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
	log.Debug().Uint16("port", port).Msg("reserve wireguard port")
	err := n.portSet.Add(uint(port))
	if err != nil {
		return errors.Wrap(err, "wireguard listen port already in use, pick another one")
	}

	return nil
}

func (n *networker) releasePort(port uint16) error {
	log.Debug().Uint16("port", port).Msg("release wireguard port")
	n.portSet.Remove(uint(port))
	return nil
}

func (n *networker) publishWGPorts() error {
	ports, err := n.portSet.List()
	if err != nil {
		return err
	}

	if err := n.tnodb.NodeSetPorts(n.identity.NodeID().Identity(), ports); err != nil {
		// maybe retry a couple of times ?
		// having bdb and the node out of sync is pretty bad
		return errors.Wrap(err, "fail to publish wireguard port to bcdb")
	}
	return nil
}

func (n *networker) getAddresses(nsName, link string) ([]netlink.Addr, error) {
	netns, err := namespace.GetByName(nsName)
	if err != nil {
		return nil, err
	}

	defer netns.Close()
	var addr []netlink.Addr
	err = netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(link)
		if err != nil {
			return err
		}
		addr, err = netlink.AddrList(link, netlink.FAMILY_ALL)
		return err
	})

	return addr, err
}

func (n *networker) monitorNS(ctx context.Context, name string, links ...string) <-chan pkg.NetlinkAddresses {
	get := func() (pkg.NetlinkAddresses, error) {
		var result pkg.NetlinkAddresses
		for _, link := range links {
			values, err := n.getAddresses(name, link)
			if err != nil {
				return nil, err
			}

			for _, value := range values {
				result = append(result, pkg.NetlinkAddress(value))
			}
		}

		return result, nil
	}

	addresses, _ := get()
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		monitorCtx, cancel := context.WithCancel(context.Background())

		defer func() {
			close(ch)
			cancel()
		}()

	main:
		for {
			updates, err := namespace.Monitor(monitorCtx, name)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					continue
				}
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-updates:
					addresses, err = get()
					if err != nil {
						// this might be duo too namespace was deleted, hence
						// we need to try find the namespace object again
						cancel()
						monitorCtx, cancel = context.WithCancel(context.Background())
						continue main
					}
				case <-time.After(2 * time.Second):
					ch <- addresses
				}
			}

		}
	}()

	return ch
}

func (n *networker) DMZAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	return n.monitorNS(ctx, ndmz.NetNSNDMZ, ndmz.DMZPub4, ndmz.DMZPub6)
}

func (n *networker) YggAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	return n.monitorNS(ctx, ndmz.NetNSNDMZ, yggdrasil.YggIface)
}

func (n *networker) PublicAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	return n.monitorNS(ctx, types.PublicNamespace, types.PublicIface)
}

func (n *networker) ZOSAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	// we don't use monitorNS because
	// 1- this is the host namespace
	// 2- the ZOS bridge must exist all the time
	updates := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribe(updates, ctx.Done()); err != nil {
		log.Fatal().Err(err).Msg("failed to listen to netlink address updates")
	}

	link, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not find the '%s' bridge", types.DefaultBridge)
	}

	get := func() pkg.NetlinkAddresses {
		var result pkg.NetlinkAddresses
		values, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
		for _, value := range values {
			result = append(result, pkg.NetlinkAddress(value))
		}

		return result
	}

	addresses := get()

	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.LinkIndex != link.Attrs().Index {
					continue
				}

				addresses = get()
			case <-time.After(2 * time.Second):
				ch <- addresses
			}
		}
	}()

	return ch

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

// resourceByNodeID return the net resource associated with a nodeID
func resourceByNodeID(nodeID string, resources []pkg.NetResource) (*pkg.NetResource, error) {
	for _, resource := range resources {
		if resource.NodeID == nodeID {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("not network resource for this node: %s", nodeID)
}

// tapName returns the name of the tap device for a network namespace
func tapName(netID pkg.NetID) (string, error) {
	name := fmt.Sprintf("t-%s", netID)
	if len(name) > 15 {
		return "", errors.Errorf("tap name too long %s", name)
	}
	return name, nil
}
