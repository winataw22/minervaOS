package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/blang/semver"

	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/macvtap"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/public"
	"github.com/threefoldtech/zos/pkg/network/tuntap"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/substrate"

	"github.com/vishvananda/netlink"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"

	"github.com/containernetworking/plugins/pkg/ns"
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
	// ZDBPubIface is pub interface name of the interface used in the 0-db network namespace
	ZDBPubIface = "zdb0"
	// ZDBYggIface is ygg interface name of the interface used in the 0-db network namespace
	ZDBYggIface = "ygg0"

	wgPortDir          = "wireguard_ports"
	networkDir         = "networks"
	ipamLeaseDir       = "ndmz-lease"
	zdbNamespacePrefix = "zdb-ns-"
)

const (
	mib = 1024 * 1024
)

var (
	//NetworkSchemaLatestVersion last version
	NetworkSchemaLatestVersion = semver.MustParse("0.1.0")
)

type networker struct {
	identity     *stubs.IdentityManagerStub
	networkDir   string
	ipamLeaseDir string
	portSet      *set.UIntSet

	publicConfig string
	ndmz         ndmz.DMZ
	ygg          *YggServer
}

var _ pkg.Networker = (*networker)(nil)

// NewNetworker create a new pkg.Networker that can be used over zbus
func NewNetworker(identity *stubs.IdentityManagerStub, publicCfgPath string, ndmz ndmz.DMZ, ygg *YggServer) (pkg.Networker, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	runtimeDir := filepath.Join(vd, networkDir)
	ipamLease := filepath.Join(vd, ipamLeaseDir)

	for _, dir := range []string{runtimeDir, ipamLease} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrapf(err, "failed to create directory: '%s'", dir)
		}
	}

	nw := &networker{
		identity:     identity,
		networkDir:   runtimeDir,
		ipamLeaseDir: ipamLease,
		portSet:      set.NewInt(),

		publicConfig: publicCfgPath,
		ygg:          ygg,
		ndmz:         ndmz,
	}

	// always add the reserved yggdrasil port to the port set so we make sure they are never
	// picked for wireguard endpoints
	for _, port := range []int{yggdrasil.YggListenTCP, yggdrasil.YggListenTLS, yggdrasil.YggListenLinkLocal} {
		if err := nw.portSet.Add(uint(port)); err != nil && errors.Is(err, set.ErrConflict{}) {
			return nil, err
		}
	}

	if err := nw.syncWGPorts(); err != nil {
		return nil, err
	}

	return nw, nil
}

var _ pkg.Networker = (*networker)(nil)

func (n *networker) Ready() error {
	return nil
}

func (n *networker) WireguardPorts() ([]uint, error) {
	return n.portSet.List()
}

// ZDBPrepare sends a macvlan interface into the
// network namespace of a ZDB container
func (n networker) ZDBPrepare(id string) (string, error) {

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte("pub:" + id))

	netNSName := zdbNamespacePrefix + strings.Replace(hw.String(), ":", "", -1)

	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", err
	}
	defer netNs.Close()

	if err := n.createMacVlan(ZDBPubIface, types.PublicBridge, hw, nil, nil, netNs); err != nil {
		return "", errors.Wrap(err, "failed to setup zdb public interface")
	}

	if n.ygg == nil {
		return netNSName, nil
	}

	// new hardware address for the ygg interface
	hw = ifaceutil.HardwareAddrFromInputBytes([]byte("ygg:" + id))

	ip, err := n.ygg.SubnetFor(hw)
	if err != nil {
		return "", fmt.Errorf("failed to generate ygg subnet IP: %w", err)
	}

	ips := []*net.IPNet{
		{
			IP:   ip,
			Mask: net.CIDRMask(64, 128),
		},
	}

	gw, err := n.ygg.Gateway()
	if err != nil {
		return "", fmt.Errorf("failed to get ygg gateway IP: %w", err)
	}

	routes := []*netlink.Route{
		{
			Dst: &net.IPNet{
				IP:   net.ParseIP("200::"),
				Mask: net.CIDRMask(7, 128),
			},
			Gw: gw.IP,
			// LinkIndex:... this is set by macvlan.Install
		},
	}

	if err := n.createMacVlan(ZDBYggIface, types.YggBridge, hw, ips, routes, netNs); err != nil {
		return "", errors.Wrap(err, "failed to setup zdb ygg interface")
	}

	return netNSName, nil
}

// ZDBDestroy is the opposite of ZDPrepare, it makes sure network setup done
// for zdb is rewind. ns param is the namespace return by the ZDBPrepare
func (n networker) ZDBDestroy(ns string) error {
	if !strings.HasPrefix(ns, zdbNamespacePrefix) {
		return fmt.Errorf("invalid zdb namespace name '%s'", ns)
	}

	if !namespace.Exists(ns) {
		return nil
	}

	nSpace, err := namespace.GetByName(ns)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to get namespace '%s'", ns)
	}

	return namespace.Delete(nSpace)
}

func (n networker) createMacVlan(iface string, master string, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netNs ns.NetNS) error {
	var macVlan *netlink.Macvlan
	err := netNs.Do(func(_ ns.NetNS) error {
		var err error
		macVlan, err = macvlan.GetByName(iface)
		return err
	})

	if _, ok := err.(netlink.LinkNotFoundError); ok {
		macVlan, err = macvlan.Create(iface, master, netNs)

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
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
func (n *networker) SetupPrivTap(networkID pkg.NetID, name string) (string, error) {
	log.Info().Str("network-id", string(networkID)).Msg("Setting up tap interface")

	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return "", errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	netRes, err := nr.New(localNR)
	if err != nil {
		return "", errors.Wrap(err, "failed to load network resource")
	}

	bridgeName, err := netRes.BridgeName()
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace bridge")
	}

	tapIface, err := tapName(name)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	if ifaceutil.Exists(tapIface, nil) {
		return tapIface, nil
	}

	_, err = tuntap.CreateTap(tapIface, bridgeName)

	return tapIface, err
}

func (n *networker) TapExists(name string) (bool, error) {
	log.Info().Str("tap-name", string(name)).Msg("Checking if tap interface exists")

	tapIface, err := tapName(name)
	if err != nil {
		return false, errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Exists(tapIface, nil), nil
}

// RemoveTap in the network resource.
func (n *networker) RemoveTap(name string) error {
	log.Info().Str("tap-name", string(name)).Msg("Removing tap interface")

	tapIface, err := tapName(name)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

func (n *networker) PublicIPv4Support() bool {
	return n.ndmz.SupportsPubIPv4()
}

// SetupPubTap sets up a tap device in the host namespace for the public ip
// reservation id. It is hooked to the public bridge. The name of the tap
// interface is returned
func (n *networker) SetupPubTap(pubIPReservationID string) (string, error) {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Setting up public tap interface")

	if !n.ndmz.SupportsPubIPv4() {
		return "", errors.New("can't create public tap on this node")
	}

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(pubIPReservationID))
	_, err = macvtap.CreateMACvTap(tapIface, public.PublicBridge, hw)

	return tapIface, err
}

// SetupYggTap sets up a tap device in the host namespace for the yggdrasil ip
func (n *networker) SetupYggTap(name string) (tap pkg.YggdrasilTap, err error) {
	log.Info().Str("pubip-res-id", string(name)).Msg("Setting up public tap interface")

	tapIface, err := tapName(name)
	if err != nil {
		return tap, errors.Wrap(err, "could not get network namespace tap device name")
	}

	tap.Name = tapIface

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte("ygg:" + name))
	tap.HW = hw
	ip, err := n.ygg.SubnetFor(hw)
	if err != nil {
		return tap, err
	}

	tap.IP = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	gw, err := n.ygg.Gateway()
	if err != nil {
		return tap, err
	}

	tap.Gateway = gw
	if ifaceutil.Exists(tapIface, nil) {
		return tap, nil
	}

	_, err = tuntap.CreateTap(tapIface, types.YggBridge)
	return tap, err
}

// PubTapExists checks if the tap device for the public network exists already
func (n *networker) PubTapExists(pubIPReservationID string) (bool, error) {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Checking if public tap interface exists")

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return false, errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Exists(tapIface, nil), nil
}

// RemovePubTap removes the public tap device from the host namespace
// of the networkID
func (n *networker) RemovePubTap(pubIPReservationID string) error {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Removing public tap interface")

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

// SetupPubIPFilter sets up filter for this public ip
func (n *networker) SetupPubIPFilter(filterName string, iface string, ip string, ipv6 string, mac string) error {
	if n.PubIPFilterExists(filterName) {
		return nil
	}

	//TODO: use nft.Apply
	cmd := exec.Command("/bin/sh", "-c",
		fmt.Sprintf(`# add vm
# add a chain for the vm public interface in arp and bridge
nft 'add chain arp filter %[1]s'
nft 'add chain bridge filter %[1]s'

# make nft jump to vm chain
nft 'add rule arp filter input iifname "%[2]s" jump %[1]s'
nft 'add rule bridge filter forward iifname "%[2]s" jump %[1]s'

# arp rule for vm
nft 'add rule arp filter %[1]s arp operation reply arp saddr ip . arp saddr ether != { %[3]s . %[4]s } drop'

# filter on L2 fowarding of non-matching ip/mac, drop RA,dhcpv6,dhcp
nft 'add rule bridge filter %[1]s ip saddr . ether saddr != { %[3]s . %[4]s } counter drop'
nft 'add rule bridge filter %[1]s ip6 saddr . ether saddr != { %[5]s . %[4]s } counter drop'
nft 'add rule bridge filter %[1]s icmpv6 type nd-router-advert drop'
nft 'add rule bridge filter %[1]s ip6 version 6 udp sport 547 drop'
nft 'add rule bridge filter %[1]s ip version 4 udp sport 67 drop'`, filterName, iface, ip, mac, ipv6))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "could not setup firewall rules for public ip\n%s", string(output))
	}

	return nil
}

// PubIPFilterExists checks if pub ip filter
func (n *networker) PubIPFilterExists(filterName string) bool {
	cmd := exec.Command(
		"/bin/sh",
		"-c",
		fmt.Sprintf(`nft list table bridge filter | grep "chain %s"`, filterName),
	)
	err := cmd.Run()
	return err == nil
}

// RemovePubIPFilter removes the filter setted up by SetupPubIPFilter
func (n *networker) RemovePubIPFilter(filterName string) error {
	cmd := exec.Command("/bin/sh", "-c",
		fmt.Sprintf(`# in bridge table
nft 'flush chain bridge filter %[1]s'
# jump to chain rule
a=$( nft -a list table bridge filter | awk '/jump %[1]s/{ print $NF}' )
nft 'delete rule bridge filter forward handle '${a}
# chain itself
a=$( nft -a list table bridge filter | awk '/chain %[1]s/{ print $NF}' )
nft 'delete chain bridge filter handle '${a}

# in arp table
nft 'flush chain arp filter %[1]s'
# jump to chain rule
a=$( nft -a list table arp filter | awk '/jump %[1]s/{ print $NF}' )
nft 'delete rule arp filter input handle '${a}
# chain itself
a=$( nft -a list table arp filter | awk '/chain %[1]s/{ print $NF}' )
nft 'delete chain arp filter handle '${a}`, filterName))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "could not tear down firewall rules for public ip\n%s", string(output))
	}
	return nil
}

// DisconnectPubTap disconnects the public tap from the network. The interface
// itself is not removed and will need to be cleaned up later
func (n *networker) DisconnectPubTap(pubIPReservationID string) error {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Disconnecting public tap interface")

	tapIfaceName, err := pubTapName(pubIPReservationID)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	tap, err := netlink.LinkByName(tapIfaceName)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil
	} else if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.Wrap(err, "could not load tap device")
	}

	//setting the txqueue on a macvtap will prevent traffic from
	//going over the device, effectively disconnecting it.
	return netlink.LinkSetTxQLen(tap, 0)
}

// GetPublicIPv6Subnet returns the IPv6 prefix op the public subnet of the host
func (n *networker) GetPublicIPv6Subnet() (net.IPNet, error) {
	addrs, err := n.ndmz.GetIP(ndmz.FamilyV6)
	if err != nil {
		return net.IPNet{}, errors.Wrap(err, "could not get ips from ndmz")
	}

	for _, addr := range addrs {
		if addr.IP.IsGlobalUnicast() && !isULA(addr.IP) && !isYgg(addr.IP) {
			return addr, nil
		}
	}

	return net.IPNet{}, fmt.Errorf("no public ipv6 found")
}

// GetSubnet of a local network resource identified by the network ID, ipv4 and ipv6
// subnet respectively
func (n networker) GetSubnet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.Subnet.IPNet, nil
}

// GetNet of a network identified by the network ID
func (n networker) GetNet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.NetworkIPRange.IPNet, nil
}

// GetDefaultGwIP returns the IPs of the default gateways inside the network
// resource identified by the network ID on the local node, for IPv4 and IPv6
// respectively
func (n networker) GetDefaultGwIP(networkID pkg.NetID) (net.IP, net.IP, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	// only IP4 atm
	ip := localNR.Subnet.IP.To4()
	if ip == nil {
		return nil, nil, errors.New("nr subnet is not valid IPv4")
	}

	// defaut gw is currently implied to be at `x.x.x.1`
	// also a subnet in a NR is assumed to be a /24
	ip[len(ip)-1] = 1

	// ipv6 is derived from the ipv4
	return ip, nr.Convert4to6(string(networkID), ip), nil
}

// GetIPv6From4 generates an IPv6 address from a given IPv4 address in a NR
func (n networker) GetIPv6From4(networkID pkg.NetID, ip net.IP) (net.IPNet, error) {
	if ip.To4() == nil {
		return net.IPNet{}, errors.New("invalid IPv4 address")
	}
	return net.IPNet{IP: nr.Convert4to6(string(networkID), ip), Mask: net.CIDRMask(64, 128)}, nil
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
		ips = make([]net.IP, 0, len(addrs))
		for i, addr := range addrs {
			ip := addr.IP
			if ip6 := ip.To16(); ip6 != nil {
				// ipv6
				if !ip6.IsGlobalUnicast() || ifaceutil.IsULA(ip6) {
					// skip if not global or is ula address
					continue
				}
			}

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
func (n *networker) CreateNR(netNR pkg.Network) (string, error) {
	log.Info().Str("network", string(netNR.NetID)).Msg("create network resource")

	// check if there is a reserved wireguard port for this NR already
	// or if we need to update it
	storedNR, err := n.networkOf(string(netNR.NetID))
	if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to load previous network setup")
	}

	if err == nil {
		if err := n.releasePort(storedNR.WGListenPort); err != nil {
			return "", err
		}
	}

	if err := n.reservePort(netNR.WGListenPort); err != nil {
		return "", err
	}

	netr, err := nr.New(netNR)
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

	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	wgName, err := netr.WGName()
	if err != nil {
		return "", errors.Wrap(err, "failed to get wg interface name for network resource")
	}

	log.Info().Msg("create network resource namespace")
	if err = netr.Create(); err != nil {
		return "", errors.Wrap(err, "failed to create network resource")
	}

	exists, err := netr.HasWireguard()
	if err != nil {
		return "", errors.Wrap(err, "failed to check if network resource has wireguard setup")
	}

	if !exists {
		var wg *wireguard.Wireguard
		wg, err = public.NewWireguard(wgName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create wg interface for network resource '%s'", netNR.NetID)
		}
		if err = netr.SetWireguard(wg); err != nil {
			return "", errors.Wrap(err, "failed to setup wireguard interface for network resource")
		}
	}

	if err = n.ndmz.AttachNR(string(netNR.NetID), netr, n.ipamLeaseDir); err != nil {
		return "", errors.Wrapf(err, "failed to attach network resource to DMZ bridge")
	}

	if err = netr.ConfigureWG(netNR.WGPrivateKey); err != nil {
		return "", errors.Wrap(err, "failed to configure network resource")
	}

	if err = n.storeNetwork(netNR); err != nil {
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

	writer, err := versioned.NewWriter(file, NetworkSchemaLatestVersion)
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
func (n *networker) DeleteNR(netNR pkg.Network) error {
	nr, err := nr.New(netNR)
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
	path := filepath.Join(n.networkDir, string(netNR.NetID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

// Set node public namespace config
func (n *networker) SetPublicConfig(cfg pkg.PublicConfig) error {
	id := n.identity.NodeID(context.Background())
	_, err := public.EnsurePublicSetup(id, &cfg)
	if err != nil {
		return errors.Wrap(err, "failed to apply public config")
	}

	if err := public.SavePublicConfig(n.publicConfig, &cfg); err != nil {
		return errors.Wrap(err, "failed to store public config")
	}

	// this is kinda dirty, but we need to update the node public config
	// on the block-chain. so ...
	env := environment.MustGet()
	sub, err := env.GetSubstrate()
	if err != nil {
		return errors.Wrap(err, "failed to connect to substrate")
	}
	sk := n.identity.PrivateKey(context.Background())
	identity, err := substrate.Identity(sk)
	if err != nil {
		return err
	}
	twin, err := sub.GetTwinByPubKey(identity.PublicKey)
	if err != nil {
		return errors.Wrap(err, "failed to get node twin by public key")
	}

	nodeID, err := sub.GetNodeByTwinID(twin)
	if err != nil {
		return errors.Wrap(err, "failed to get node by twin id")
	}

	node, err := sub.GetNode(nodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get node with id: %d", nodeID)
	}

	cfg, err = public.GetPublicSetup()
	if err != nil {
		return errors.Wrap(err, "failed to get public setup")
	}

	subCfg := substrate.OptionPublicConfig{
		HasValue: true,
		AsValue: substrate.PublicConfig{
			IPv4: cfg.IPv4.String(),
			IPv6: cfg.IPv6.String(),
			GWv4: cfg.GW4.String(),
			GWv6: cfg.GW6.String(),
		},
	}

	if reflect.DeepEqual(node.PublicConfig, subCfg) {
		//nothing to do
		return nil
	}
	// update the node
	node.PublicConfig = subCfg
	_, err = sub.UpdateNode(sk, *node)
	return err
}

// Get node public namespace config
func (n *networker) GetPublicConfig() (pkg.PublicConfig, error) {
	// TODO: instea of loading, this actually must get
	// from reality.
	cfg, err := public.GetPublicSetup()
	if err != nil {
		return pkg.PublicConfig{}, err
	}
	return cfg, nil
}

func (n *networker) networkOf(id string) (nr pkg.Network, err error) {
	path := filepath.Join(n.networkDir, string(id))
	file, err := os.OpenFile(path, os.O_RDWR, 0660)
	if err != nil {
		return nr, err
	}
	defer file.Close()

	reader, err := versioned.NewReader(file)
	if versioned.IsNotVersioned(err) {
		// old data that doesn't have any version information
		if _, err := file.Seek(0, 0); err != nil {
			return nr, err
		}

		reader = versioned.NewVersionedReader(NetworkSchemaLatestVersion, file)
	} else if err != nil {
		return nr, err
	}

	var net pkg.Network
	dec := json.NewDecoder(reader)

	version := reader.Version()
	//validV1 := versioned.MustParseRange(fmt.Sprintf("=%s", pkg.NetworkSchemaV1))
	validLatest := versioned.MustParseRange(fmt.Sprintf("<=%s", NetworkSchemaLatestVersion.String()))

	if validLatest(version) {
		if err := dec.Decode(&net); err != nil {
			return nr, err
		}
	} else {
		return nr, fmt.Errorf("unknown network object version (%s)", version)
	}

	return net, nil
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

func (n *networker) DMZAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := n.ndmz.GetIP(ndmz.FamilyAll)
				if err != nil {
					log.Error().Err(err).Msg("failed to get dmz IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
}

func (n *networker) YggAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := n.ndmz.GetIPFor(yggdrasil.YggIface)
				if err != nil {
					log.Error().Err(err).Str("inf", yggdrasil.YggIface).Msg("failed to get public IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
}

func (n *networker) PublicAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := public.IPs()
				if err != nil {
					log.Error().Err(err).Msg("failed to get public IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
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
			result = append(result, *value.IPNet)
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

func (n *networker) syncWGPorts() error {
	names, err := namespace.List("n-")
	if err != nil {
		return err
	}

	readPort := func(name string) (int, error) {
		netNS, err := namespace.GetByName(name)
		if err != nil {
			return 0, err
		}
		defer netNS.Close()

		ifaceName := strings.Replace(name, "n-", "w-", 1)

		var port int
		err = netNS.Do(func(_ ns.NetNS) error {
			link, err := wireguard.GetByName(ifaceName)
			if err != nil {
				return err
			}
			d, err := link.Device()
			if err != nil {
				return err
			}

			port = d.ListenPort
			return nil
		})
		if err != nil {
			return 0, err
		}

		return port, nil
	}

	for _, name := range names {
		port, err := readPort(name)
		if err != nil {
			log.Error().Err(err).Str("namespace", name).Msgf("failed to read port for network namespace")
			continue
		}
		//skip error cause we don't care if there are some duplicate at this point
		_ = n.portSet.Add(uint(port))
	}

	return nil
}

// createNetNS create a network namespace and set lo interface up
func createNetNS(name string) (ns.NetNS, error) {
	var netNs ns.NetNS
	var err error
	if namespace.Exists(name) {
		netNs, err = namespace.GetByName(name)
	} else {
		netNs, err = namespace.Create(name)
	}

	if err != nil {
		return nil, fmt.Errorf("fail to create network namespace %s: %w", name, err)
	}

	err = netNs.Do(func(_ ns.NetNS) error {
		return ifaceutil.SetLoUp()
	})

	if err != nil {
		namespace.Delete(netNs)
		return nil, fmt.Errorf("failed to bring lo interface up in namespace %s: %w", name, err)
	}

	return netNs, nil
}

// tapName prefixes the tap name with a t-
func tapName(tname string) (string, error) {
	name := fmt.Sprintf("t-%s", tname)
	if len(name) > 15 {
		return "", errors.Errorf("tap name too long %s", name)
	}
	return name, nil
}

func pubTapName(resID string) (string, error) {
	name := fmt.Sprintf("p-%s", resID)
	if len(name) > 15 {
		return "", errors.Errorf("tap name too long %s", name)
	}
	return name, nil
}

var ulaPrefix = net.IPNet{
	IP:   net.ParseIP("fc00::"),
	Mask: net.CIDRMask(7, 128),
}

func isULA(ip net.IP) bool {
	return ulaPrefix.Contains(ip)
}

var yggPrefix = net.IPNet{
	IP:   net.ParseIP("200::"),
	Mask: net.CIDRMask(7, 128),
}

func isYgg(ip net.IP) bool {
	return yggPrefix.Contains(ip)
}
