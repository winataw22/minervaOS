package network

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"

	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/iperf"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/public"
	"github.com/threefoldtech/zos/pkg/network/tuntap"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"

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
	// PubIface is pub interface name of the interface used in the 0-db network namespace
	PubIface = "eth0"
	// ZDBYggIface is ygg interface name of the interface used in the 0-db network namespace
	ZDBYggIface = "ygg0"

	networkDir          = "networks"
	linkDir             = "link"
	ipamLeaseDir        = "ndmz-lease"
	myceliumKeyDir      = "mycelium-key"
	zdbNamespacePrefix  = "zdb-ns-"
	qsfsNamespacePrefix = "qfs-ns-"
)

const (
	mib = 1024 * 1024
)

var (
	//NetworkSchemaLatestVersion last version
	NetworkSchemaLatestVersion = semver.MustParse("0.1.0")
)

type networker struct {
	identity       *stubs.IdentityManagerStub
	networkDir     string
	linkDir        string
	ipamLeaseDir   string
	myceliumKeyDir string
	portSet        *set.UIntSet

	ndmz ndmz.DMZ
	ygg  *yggdrasil.YggServer
}

var _ pkg.Networker = (*networker)(nil)

// NewNetworker create a new pkg.Networker that can be used over zbus
func NewNetworker(identity *stubs.IdentityManagerStub, ndmz ndmz.DMZ, ygg *yggdrasil.YggServer) (pkg.Networker, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	runtimeDir := filepath.Join(vd, networkDir)
	linkDir := filepath.Join(runtimeDir, linkDir)
	ipamLease := filepath.Join(vd, ipamLeaseDir)
	myceliumKey := filepath.Join(vd, myceliumKeyDir)

	for _, dir := range []string{linkDir, ipamLease, myceliumKey} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrapf(err, "failed to create directory: '%s'", dir)
		}
	}

	nw := &networker{
		identity:       identity,
		networkDir:     runtimeDir,
		linkDir:        linkDir,
		ipamLeaseDir:   ipamLease,
		myceliumKeyDir: myceliumKey,
		portSet:        set.NewInt(),

		ygg:  ygg,
		ndmz: ndmz,
	}

	// always add the reserved yggdrasil port to the port set so we make sure they are never
	// picked for wireguard endpoints
	// we also add http, https, and traefik metrics ports 8082 to the list.
	for _, port := range []int{yggdrasil.YggListenTCP, yggdrasil.YggListenTLS, yggdrasil.YggListenLinkLocal, iperf.IperfPort, 80, 443, 8082} {
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

func (n *networker) attachYgg(id string, netNs ns.NetNS) (net.IPNet, error) {
	// new hardware address for the ygg interface
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte("ygg:" + id))

	ip, err := n.ygg.SubnetFor(hw)
	if err != nil {
		return net.IPNet{}, fmt.Errorf("failed to generate ygg subnet IP: %w", err)
	}

	ips := []*net.IPNet{
		&ip,
	}

	gw, err := n.ygg.Gateway()
	if err != nil {
		return net.IPNet{}, fmt.Errorf("failed to get ygg gateway IP: %w", err)
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
		return net.IPNet{}, errors.Wrap(err, "failed to setup zdb ygg interface")
	}

	return ip, nil
}

func (n *networker) detachYgg(id string, netNs ns.NetNS) error {
	return netNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(ZDBYggIface)
		if err != nil {
			return err
		}
		if err := netlink.LinkDel(link); err != nil {
			return errors.Wrap(err, "failed to delete zdb ygg interface")
		}
		return nil
	})
}

// prepare creates a unique namespace (based on id) with "prefix"
// and make sure it's wired to the bridge on host namespace
func (n *networker) prepare(id, prefix, bridge string) (string, error) {
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte("pub:" + id))

	netNSName := prefix + strings.Replace(hw.String(), ":", "", -1)

	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", err
	}
	defer netNs.Close()

	if err := n.createMacVlan(PubIface, bridge, hw, nil, nil, netNs); err != nil {
		return "", errors.Wrap(err, "failed to setup zdb public interface")
	}

	if n.ygg == nil {
		return netNSName, nil
	}
	_, err = n.attachYgg(id, netNs)
	return netNSName, err
}

func (n *networker) destroy(ns string) error {
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

// func (n *networker) NSPrepare(id string, )
// ZDBPrepare sends a macvlan interface into the
// network namespace of a ZDB container
func (n *networker) ZDBPrepare(id string) (string, error) {
	return n.prepare(id, zdbNamespacePrefix, types.PublicBridge)
}

// ZDBDestroy is the opposite of ZDPrepare, it makes sure network setup done
// for zdb is rewind. ns param is the namespace return by the ZDBPrepare
func (n *networker) ZDBDestroy(ns string) error {
	panic("not implemented")
	// if !strings.HasPrefix(ns, zdbNamespacePrefix) {
	// 	return fmt.Errorf("invalid zdb namespace name '%s'", ns)
	// }

	// return n.destroy(ns)
}

func (n *networker) createMacVlan(iface string, master string, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netNs ns.NetNS) error {
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
func (n *networker) SetupPrivTap(networkID pkg.NetID, name string) (ifc string, err error) {
	log.Info().Str("network-id", string(networkID)).Msg("Setting up tap interface")

	localNR, err := n.networkOf(networkID)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	netRes := nr.New(localNR, n.myceliumKeyDir)

	bridgeName, err := netRes.BridgeName()
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace bridge")
	}

	tapIface, err := tapName(name)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
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
func (n *networker) SetupPubTap(name string) (string, error) {
	log.Info().Str("pubtap-name", string(name)).Msg("Setting up public tap interface")

	if !n.ndmz.SupportsPubIPv4() {
		return "", errors.New("can't create public tap on this node")
	}

	tapIface, err := pubTapName(name)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	_, err = tuntap.CreateTap(tapIface, public.PublicBridge)

	return tapIface, err
}

// SetupMyceliumTap creates a new mycelium tap device attached to this network resource with deterministic IP address
func (n *networker) SetupMyceliumTap(name string, netID zos.NetID, config zos.MyceliumIP) (tap pkg.PlanetaryTap, err error) {
	log.Info().Str("tap-name", string(name)).Msg("Setting up mycelium tap interface")

	network, err := n.networkOf(netID)
	if err != nil {
		return tap, errors.Wrapf(err, "failed to get network resource '%s'", netID)
	}

	if network.Mycelium == nil {
		return tap, fmt.Errorf("network resource does not support mycelium")
	}

	tapIface, err := tapName(name)
	if err != nil {
		return tap, errors.Wrap(err, "could not get network namespace tap device name")
	}

	tap.Name = tapIface

	// calculate the hw address that will be set INSIDE the vm (not the host)
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte("mycelium:" + name))
	tap.HW = hw

	netNR := nr.New(network, n.myceliumKeyDir)

	ip, gw, err := netNR.MyceliumIP(config.Seed)
	if err != nil {
		return tap, err
	}
	tap.IP = ip
	tap.Gateway = gw

	if ifaceutil.Exists(tapIface, nil) {
		return tap, nil
	}

	if err := netNR.AttachMycelium(tapIface); err != nil {
		return tap, err
	}

	return tap, err
}

// SetupYggTap sets up a tap device in the host namespace for the yggdrasil ip
func (n *networker) SetupYggTap(name string) (tap pkg.PlanetaryTap, err error) {
	log.Info().Str("tap-name", string(name)).Msg("Setting up yggdrasil tap interface")

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

	tap.IP = ip

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
func (n *networker) PubTapExists(name string) (bool, error) {
	log.Info().Str("pubtap-name", string(name)).Msg("Checking if public tap interface exists")

	tapIface, err := pubTapName(name)
	if err != nil {
		return false, errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Exists(tapIface, nil), nil
}

// RemovePubTap removes the public tap device from the host namespace
// of the networkID
func (n *networker) RemovePubTap(name string) error {
	log.Info().Str("pubtap-name", string(name)).Msg("Removing public tap interface")

	tapIface, err := pubTapName(name)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

var (
	pubIpTemplateSetup = template.Must(template.New("filter-setup").Parse(
		`# add vm
# add a chain for the vm public interface in arp and bridge
nft 'add chain bridge filter {{.Name}}-pre'
nft 'add chain bridge filter {{.Name}}-post'

# make nft jump to vm chain
nft 'add rule bridge filter prerouting iifname "{{.Iface}}" jump {{.Name}}-pre'
nft 'add rule bridge filter postrouting oifname "{{.Iface}}" jump {{.Name}}-post'

nft 'add rule bridge filter {{.Name}}-pre ip saddr . ether saddr != { {{.IPv4}} . {{.Mac}} } counter drop'
# nft 'add rule bridge filter {{.Name}}-pre ip6 saddr . ether saddr != { {{.IPv6}} . {{.Mac}} } counter drop'

nft 'add rule bridge filter {{.Name}}-pre arp operation reply arp saddr ip != {{.IPv4}} counter drop'
nft 'add rule bridge filter {{.Name}}-pre arp operation request arp saddr ip != {{.IPv4}} counter drop'

nft 'add rule bridge filter {{.Name}}-post ip daddr . ether daddr != { {{.IPv4}} . {{.Mac}} } counter drop'
# nft 'add rule bridge filter {{.Name}}-post ip6 saddr . ether saddr != { {{.IPv6}} . {{.Mac}} } counter drop'
`))

	pubIpTemplateDestroy = template.Must(template.New("filter-destroy").Parse(
		`# in bridge table
nft 'flush chain bridge filter {{.Name}}-post'
nft 'flush chain bridge filter {{.Name}}-pre'

# the .name rule is for backward compatibility
# to make sure older chains are deleted
nft 'flush chain bridge filter {{.Name}}' || true

# we need to make sure this clean up can also work on older setup
# jump to chain rule
a=$( nft -a list table bridge filter | awk '/jump {{.Name}}-pre/{ print $NF}' )
if [ -n "${a}" ]; then
	nft delete rule bridge filter prerouting handle ${a}
fi
a=$( nft -a list table bridge filter | awk '/jump {{.Name}}-post/{ print $NF}' )
if [ -n "${a}" ]; then
	nft delete rule bridge filter postrouting handle ${a}
fi
a=$( nft -a list table bridge filter | awk '/jump {{.Name}}/{ print $NF}' )
if [ -n "${a}" ]; then
	nft delete rule bridge filter forward handle ${a}
fi

# chain itself
for chain in $( nft -a list table bridge filter | awk '/chain {{.Name}}/{ print $NF}' ); do
	nft delete chain bridge filter handle ${chain}
done

# the next section is only for backward compatibility
# in arp table
nft 'flush chain arp filter {{.Name}}'
# jump to chain rule
a=$( nft -a list table arp filter | awk '/jump {{.Name}}/{ print $NF}' )
if [ -n "${a}" ]; then
	nft delete rule arp filter input handle ${a}
fi
# chain itself
a=$( nft -a list table arp filter | awk '/chain {{.Name}}/{ print $NF}' )
if [ -n "${a}" ]; then
	nft delete chain arp filter handle ${a}
fi
`))
)

// SetupPubIPFilter sets up filter for this public ip
func (n *networker) SetupPubIPFilter(filterName string, iface string, ipv4 net.IP, ipv6 net.IP, mac string) error {
	if n.PubIPFilterExists(filterName) {
		return nil
	}

	ipv4 = ipv4.To4()
	ipv6 = ipv6.To16()
	// if no ipv4 or ipv6 provided, we make sure
	// to use zero ip so the user can't just assign
	// an ip to his vm to use.
	if len(ipv4) == 0 {
		ipv4 = net.IPv4zero
	}

	if len(ipv6) == 0 {
		ipv6 = net.IPv6zero
	}

	data := struct {
		Name  string
		Iface string
		Mac   string
		IPv4  string
		IPv6  string
	}{
		Name:  filterName,
		Iface: iface,
		Mac:   mac,
		IPv4:  ipv4.String(),
		IPv6:  ipv6.String(),
	}

	var buffer bytes.Buffer
	if err := pubIpTemplateSetup.Execute(&buffer, data); err != nil {
		return errors.Wrap(err, "failed to execute filter template")
	}

	//TODO: use nft.Apply
	cmd := exec.Command("/bin/sh", "-c", buffer.String())

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
	data := struct {
		Name string
	}{
		Name: filterName,
	}

	var buffer bytes.Buffer
	if err := pubIpTemplateDestroy.Execute(&buffer, data); err != nil {
		return errors.Wrap(err, "failed to execute filter template")
	}

	cmd := exec.Command("/bin/sh", "-c", buffer.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "could not tear down firewall rules for public ip\n%s", string(output))
	}
	return nil
}

// DisconnectPubTap disconnects the public tap from the network. The interface
// itself is not removed and will need to be cleaned up later
func (n *networker) DisconnectPubTap(name string) error {
	log.Info().Str("pubtap-name", string(name)).Msg("Disconnecting public tap interface")
	tapIfaceName, err := pubTapName(name)
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

	return netlink.LinkSetNoMaster(tap)
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

func (n *networker) GetPublicIPV6Gateway() (net.IP, error) {
	// simply find the default gw for a well known public ip. in this case
	// we use the public google dns service
	return n.ndmz.GetDefaultGateway(net.ParseIP("2001:4860:4860::8888"))
}

// GetSubnet of a local network resource identified by the network ID, ipv4 and ipv6
// subnet respectively
func (n *networker) GetSubnet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(networkID)
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.Subnet.IPNet, nil
}

// GetNet of a network identified by the network ID
func (n *networker) GetNet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(networkID)
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.NetworkIPRange.IPNet, nil
}

// GetDefaultGwIP returns the IPs of the default gateways inside the network
// resource identified by the network ID on the local node, for IPv4 and IPv6
// respectively
func (n *networker) GetDefaultGwIP(networkID pkg.NetID) (net.IP, net.IP, error) {
	localNR, err := n.networkOf(networkID)
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
func (n *networker) GetIPv6From4(networkID pkg.NetID, ip net.IP) (net.IPNet, error) {
	if ip.To4() == nil {
		return net.IPNet{}, errors.New("invalid IPv4 address")
	}
	return net.IPNet{IP: nr.Convert4to6(string(networkID), ip), Mask: net.CIDRMask(64, 128)}, nil
}

func (n *networker) SetPublicExitDevice(iface string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}

	return public.SetPublicExitLink(link)
}

func (n *networker) Interfaces(iface string, netns string) (map[string]pkg.Interface, error) {
	getter := func(iface string) ([]netlink.Link, error) {
		if iface != "" {
			l, err := netlink.LinkByName(iface)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get interface %s", iface)
			}
			return []netlink.Link{l}, nil
		}

		all, err := netlink.LinkList()
		if err != nil {
			return nil, err
		}
		filtered := all[:0]
		for _, l := range all {
			name := l.Attrs().Name

			if name == "lo" ||
				(l.Type() != "device" && name != types.DefaultBridge) {

				continue
			}

			filtered = append(filtered, l)
		}

		return filtered, nil
	}

	interfaces := make(map[string]pkg.Interface)
	f := func(_ ns.NetNS) error {
		links, err := getter(iface)
		if err != nil {
			return errors.Wrapf(err, "failed to get interfaces (query: '%s')", iface)
		}

		for _, link := range links {

			addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
			if err != nil {
				return errors.Wrapf(err, "failed to list addresses of interfaces %s", iface)
			}
			ips := make([]net.IPNet, 0, len(addrs))
			for _, addr := range addrs {
				ip := addr.IP
				if ip6 := ip.To16(); ip6 != nil {
					// ipv6
					if !ip6.IsGlobalUnicast() || ifaceutil.IsULA(ip6) {
						// skip if not global or is ula address
						continue
					}
				}

				ips = append(ips, *addr.IPNet)
			}

			interfaces[link.Attrs().Name] = pkg.Interface{
				Name: link.Attrs().Name,
				Mac:  link.Attrs().HardwareAddr.String(),
				IPs:  ips,
			}
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

	return interfaces, nil
}

// [obsolete] use Interfaces instead Addrs return the IP addresses of interface
func (n *networker) Addrs(iface string, netns string) (ips []net.IP, mac string, err error) {
	if iface == "" {
		return ips, mac, fmt.Errorf("iface cannot be empty")
	}
	interfaces, err := n.Interfaces(iface, netns)
	if err != nil {
		return nil, "", err
	}

	inf := interfaces[iface]
	mac = inf.Mac
	for _, ip := range inf.IPs {
		ips = append(ips, ip.IP)
	}
	return
}

// CreateNR implements pkg.Networker interface
func (n *networker) CreateNR(wl gridtypes.WorkloadID, netNR pkg.Network) (string, error) {
	log.Info().Str("network", string(netNR.NetID)).Msg("create network resource")

	if err := n.storeNetwork(wl, netNR); err != nil {
		return "", errors.Wrap(err, "failed to store network object")
	}

	// check if there is a reserved wireguard port for this NR already
	// or if we need to update it
	storedNR, err := n.networkOf(netNR.NetID)
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

	netr := nr.New(netNR, n.myceliumKeyDir)

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

	// setup mycelium
	if err := netr.SetMycelium(); err != nil {
		return "", errors.Wrap(err, "failed to setup mycelium")
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

	nsName, err := netr.Namespace()
	if err != nil {
		return "", errors.Wrap(err, "failed to get network resource namespace")
	}

	if err = n.ndmz.AttachNR(string(netNR.NetID), nsName, n.ipamLeaseDir); err != nil {
		return "", errors.Wrapf(err, "failed to attach network resource to DMZ bridge")
	}

	if err = netr.ConfigureWG(netNR.WGPrivateKey); err != nil {
		return "", errors.Wrap(err, "failed to configure network resource")
	}

	return netr.Namespace()
}

func (n *networker) rmNetwork(wl gridtypes.WorkloadID) error {
	netID, err := zos.NetworkIDFromWorkloadID(wl)
	if err != nil {
		return err
	}

	rm := []string{
		filepath.Join(n.networkDir, netID.String()),
		filepath.Join(n.linkDir, wl.String()),
	}

	for _, p := range rm {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.Error().Err(err).Str("path", p).Msg("failed to delete file")
		}
	}

	return nil
}

func (n *networker) storeNetwork(wl gridtypes.WorkloadID, network pkg.Network) error {
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
	link := filepath.Join(n.linkDir, wl.String())
	if err := os.Symlink(filepath.Join("../", string(network.NetID)), link); err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "failed to create network symlink")
	}
	return nil
}

// DeleteNR implements pkg.Networker interface
func (n *networker) DeleteNR(wl gridtypes.WorkloadID) error {
	netID, err := zos.NetworkIDFromWorkloadID(wl)
	if err != nil {
		return err
	}
	netNR, err := n.networkOf(netID)
	if err != nil {
		return err
	}

	nr := nr.New(netNR, n.myceliumKeyDir)

	if err := nr.Delete(); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}

	if err := n.releasePort(netNR.WGListenPort); err != nil {
		log.Error().Err(err).Msg("release wireguard port failed")
		// TODO: should we return the error ?
	}

	if err := n.ndmz.DetachNR(string(netNR.NetID), n.ipamLeaseDir); err != nil {
		log.Error().Err(err).Msg("failed to detach network from ndmz")
	}

	if err := n.rmNetwork(wl); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) Namespace(id zos.NetID) string {
	return fmt.Sprintf("n-%s", id)
}

func (n *networker) UnsetPublicConfig() error {
	id := n.identity.NodeID(context.Background())
	_, err := public.EnsurePublicSetup(id, environment.MustGet().PubVlan, nil)
	return err
}

// Set node public namespace config
func (n *networker) SetPublicConfig(cfg pkg.PublicConfig) error {
	if cfg.Equal(pkg.PublicConfig{}) {
		return fmt.Errorf("public config cannot be unset, only modified")
	}

	current, err := public.LoadPublicConfig()
	if err != nil && err != public.ErrNoPublicConfig {
		return errors.Wrapf(err, "failed to load current public configuration")
	}

	if current != nil && current.Equal(cfg) {
		//nothing to do
		return nil
	}

	id := n.identity.NodeID(context.Background())
	_, err = public.EnsurePublicSetup(id, environment.MustGet().PubVlan, &cfg)
	if err != nil {
		return errors.Wrap(err, "failed to apply public config")
	}

	if err := public.SavePublicConfig(cfg); err != nil {
		return errors.Wrap(err, "failed to store public config")
	}

	// when public setup is updated. it can take a while but the capacityd
	// will detect this change and take necessary actions to update the node
	ctx := context.Background()
	sk := ed25519.PrivateKey(n.identity.PrivateKey(ctx))
	ns, err := yggdrasil.NewYggdrasilNamespace(public.PublicNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to setup public namespace for yggdrasil")
	}
	ygg, err := yggdrasil.EnsureYggdrasil(context.Background(), sk, ns)
	if err != nil {
		return err
	}

	// since the public ip might have changed, it seems sometimes ygg needs to be
	// restart to use new public ip
	if err := ygg.Restart(zinit.Default()); err != nil {
		log.Error().Err(err).Msg("failed to restart yggdrasil service")
	}

	// if yggdrasil is living inside public namespace
	// we still need to setup ndmz to also have yggdrasil but we set the yggdrasil interface
	// a different Ip that lives inside the yggdrasil range.
	dmzYgg, err := yggdrasil.NewYggdrasilNamespace(n.ndmz.Namespace())
	if err != nil {
		return errors.Wrap(err, "failed to setup ygg for dmz namespace")
	}

	ip, err := ygg.SubnetFor([]byte(fmt.Sprintf("ygg:%s", n.ndmz.Namespace())))
	if err != nil {
		return errors.Wrap(err, "failed to calculate ip for ygg inside dmz")
	}

	gw, err := ygg.Gateway()
	if err != nil {
		return err
	}

	if err := dmzYgg.SetYggIP(ip, gw.IP); err != nil {
		return errors.Wrap(err, "failed to set yggdrasil ip for dmz")
	}

	return nil
}

func (n *networker) GetPublicExitDevice() (pkg.ExitDevice, error) {
	exit, err := public.GetCurrentPublicExitLink()
	if err != nil {
		return pkg.ExitDevice{}, err
	}

	// if exit is over veth then we going over zos bridge
	// hence it's a single nic setup
	if ok, _ := bootstrap.VEthFilter(exit); ok {
		return pkg.ExitDevice{IsSingle: true}, nil
	}

	return pkg.ExitDevice{IsDual: true, AsDualInterface: exit.Attrs().Name}, nil
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

func (n *networker) networkOf(id zos.NetID) (nr pkg.Network, err error) {
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

func (n *networker) Metrics() (pkg.NetResourceMetrics, error) {
	links, err := os.ReadDir(n.linkDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list networks")
	}

	metrics := make(pkg.NetResourceMetrics)
	for _, link := range links {
		if link.IsDir() {
			continue
		}

		wl := link.Name()
		logger := log.With().Str("workload", wl).Logger()
		sym, err := os.Readlink(filepath.Join(n.linkDir, wl))
		if err != nil {
			logger.Error().Err(err).Msg("failed to get network name from workload link")
			continue
		}
		nsName := n.Namespace(zos.NetID(filepath.Base(sym)))
		logger.Debug().Str("namespace", nsName).Msg("collecting namespace statistics")
		nr, err := namespace.GetByName(nsName)
		if err != nil {
			// this happens on some node. it's weird the the namespace is suddenly gone
			// while the workload is still active.
			// TODO: investigate
			// Note: I set it to debug because it shows error in logs of logs
			logger.Debug().
				Str("namespace", nsName).
				Err(err).
				Msg("failed to get network namespace from workload")
			continue
		}

		defer nr.Close()
		err = nr.Do(func(_ ns.NetNS) error {
			// get stats of public interface.
			m, err := metricsForNics("public")
			if err != nil {
				return err
			}

			metrics[wl] = m
			return nil
		})

		if err != nil {
			log.Error().Err(err).Msg("failed to collect metrics for network")
		}
	}

	return metrics, nil
}

func (n *networker) YggAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := n.ndmz.GetIPFor(yggdrasil.YggNSInf)
				if err != nil {
					log.Error().Err(err).Str("inf", yggdrasil.YggIface).Msg("failed to get public IPs")
				}
				filtered := ips[:0]
				for _, ip := range ips {
					if yggdrasil.YggRange.Contains(ip.IP) {
						filtered = append(filtered, ip)
					}
				}
				ch <- filtered
			}
		}
	}()

	return ch
}

func (n *networker) PublicAddresses(ctx context.Context) <-chan pkg.OptionPublicConfig {
	ch := make(chan pkg.OptionPublicConfig)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				cfg, err := n.GetPublicConfig()
				ch <- pkg.OptionPublicConfig{
					PublicConfig:    cfg,
					HasPublicConfig: err == nil,
				}
			}
		}
	}()

	return ch
}

func (n *networker) ZOSAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	get := func() pkg.NetlinkAddresses {
		var result pkg.NetlinkAddresses
		link, err := netlink.LinkByName(types.DefaultBridge)
		if err != nil {
			log.Error().Err(err).Msgf("could not find the '%s' bridge", types.DefaultBridge)
			return nil
		}
		values, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.Error().Err(err).Msgf("could not list the '%s' bridge ips", types.DefaultBridge)
			return nil
		}
		for _, value := range values {
			result = append(result, *value.IPNet)
		}

		return result
	}

	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ch <- get()
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
		_ = namespace.Delete(netNs)
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

func metricsForNics(nics ...string) (m pkg.NetMetric, err error) {
	for _, nic := range nics {
		l, err := netlink.LinkByName(nic)
		if err != nil {
			return pkg.NetMetric{}, err
		}

		stats := l.Attrs().Statistics
		m.NetRxBytes += stats.RxBytes
		m.NetTxBytes += stats.TxBytes
		m.NetRxPackets += stats.RxPackets
		m.NetTxPackets += stats.TxPackets
	}

	return
}
