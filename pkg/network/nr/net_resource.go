package nr

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/threefoldtech/zos/pkg/network/tuntap"
	"github.com/threefoldtech/zos/pkg/zinit"

	mapset "github.com/deckarep/golang-set"

	"github.com/pkg/errors"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
	"github.com/vishvananda/netlink"
)

const (
	myceliumInterfaceName = "br-my"
)

var (
	myceliumIpBase = []byte{
		0xff, 0x0f,
	}

	invalidMyceliumSeeds = [][]byte{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
)

type MyceliumInspection struct {
	PublicHexKey string `json:"publicKey"`
	Address      net.IP `json:"address"`
}

// Gateway derive the gateway IP from the mycelium IP in the /64 range. It also
// return the full /64 subnet.
func (m *MyceliumInspection) Gateway() (subnet net.IPNet, gw net.IPNet, err error) {
	// here we need to return 2 things:
	// - the IP range /64 for that IP
	// - the gw address for that /64 range

	ipv6 := m.Address.To16()
	if ipv6 == nil {
		return gw, subnet, fmt.Errorf("invalid mycelium ip")
	}

	ip := make(net.IP, net.IPv6len)
	copy(ip[0:8], ipv6[0:8])

	subnet = net.IPNet{
		IP:   make(net.IP, net.IPv6len),
		Mask: net.CIDRMask(64, 128),
	}
	copy(subnet.IP, ip)

	ip[net.IPv6len-1] = 1

	gw = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	return
}

func (m *MyceliumInspection) IP(seed zos.Bytes) (ip net.IPNet, gw net.IPNet, err error) {

	if slices.ContainsFunc(invalidMyceliumSeeds, func(b []byte) bool {
		return slices.Equal(seed, b)
	}) {
		return ip, gw, fmt.Errorf("invalid seed")
	}

	// first find the base subnet.
	ip, gw, err = m.Gateway()
	if err != nil {
		return ip, gw, err
	}

	// the subnet already have the /64 part of the network (that's 8 bytes)
	// we then add a fixed 2 bytes this will avoid reusing the same gw or
	// the device ip
	copy(ip.IP[8:10], myceliumIpBase)
	// then finally we use the 6 bytes seed to build the rest of the IP
	copy(ip.IP[10:16], seed)

	return
}

// NetResource holds the logic to configure an network resource
type NetResource struct {
	id pkg.NetID
	// local network resources
	resource pkg.Network
	// network IP range, usually a /16
	networkIPRange net.IPNet

	// keyDir location where keys can be stored
	keyDir string
}

// New creates a new NetResource object
// iprange is the full network subnet
// keyDir is the path where keys (mainly mycelium)
// is stored.
func New(nr pkg.Network, keyDir string) *NetResource {
	return &NetResource{
		//fix here
		id:             nr.NetID,
		resource:       nr,
		networkIPRange: nr.NetworkIPRange.IPNet,
		keyDir:         keyDir,
	}
}

func (nr *NetResource) String() string {
	b, err := json.Marshal(nr.resource)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ID returns the network ID in which the NetResource is defined
func (nr *NetResource) ID() string {
	return string(nr.id)
}

// BridgeName returns the name of the bridge to create for the network
// resource in the host network namespace
func (nr *NetResource) BridgeName() (string, error) {
	name := fmt.Sprintf("b-%s", nr.id)
	if len(name) > 15 {
		return "", errors.Errorf("bridge namespace too long %s", name)
	}
	return name, nil
}

func (nr *NetResource) myceliumBridgeName() (string, error) {
	name := fmt.Sprintf("m-%s", nr.id)
	if len(name) > 15 {
		return "", errors.Errorf("bridge namespace too long %s", name)
	}
	return name, nil
}

// Namespace returns the name of the network namespace to create for the network resource
func (nr *NetResource) Namespace() (string, error) {
	name := fmt.Sprintf("n-%s", nr.id)
	if len(name) > 15 {
		return "", errors.Errorf("network namespace too long %s", name)
	}
	return name, nil
}

// NRIface returns name of netresource local interface
func (nr *NetResource) NRIface() (string, error) {
	name := fmt.Sprintf("n-%s", nr.id)
	if len(name) > 15 {
		return "", errors.Errorf("NR interface name too long %s", name)
	}
	return name, nil
}

// WGName returns the name of the wireguard interface to create for the network resource
func (nr *NetResource) WGName() (string, error) {
	wgName := fmt.Sprintf("w-%s", nr.id)
	if len(wgName) > 15 {
		return "", errors.Errorf("network namespace too long %s", wgName)
	}
	return wgName, nil
}

// Create setup the basic components of the network resource
// network namespace, bridge, wireguard interface and veth pair
func (nr *NetResource) Create() error {
	log.Debug().Str("nr", nr.String()).Msg("create network resource")

	if err := nr.ensureNRBridge(); err != nil {
		return err
	}
	if err := nr.createNetNS(); err != nil {
		return err
	}
	if err := nr.attachToNRBridge(); err != nil {
		return err
	}

	if err := nr.applyFirewall(); err != nil {
		return err
	}

	return nil
}

func (nr *NetResource) myceliumServiceName() string {
	return fmt.Sprintf("mycelium-%s", nr.ID())
}

func (nr *NetResource) MyceliumIP(seed zos.Bytes) (ip net.IPNet, gw net.IPNet, err error) {
	if len(seed) != zos.MyceliumIPSeedLen {
		return ip, gw, fmt.Errorf("invalid mycelium seed length")
	}

	mycelium, err := nr.inspectMycelium(filepath.Join(nr.keyDir, nr.ID()))
	if os.IsNotExist(err) {
		return ip, gw, fmt.Errorf("mycelium is not configured for this network resource")
	} else if err != nil {
		return ip, gw, err
	}

	return mycelium.IP(seed)
}

// AttachMycelium attaches a tap device to mycelium, move it to the host namespace
// to it can be used by VMs later.
// It also return the IP that should be used with the interface
func (nr *NetResource) AttachMycelium(name string) (err error) {
	brName, err := nr.myceliumBridgeName()
	if err != nil {
		return err
	}

	_, err = tuntap.CreateTap(name, brName)
	if err != nil {
		return err
	}

	return nil
}

func (nr *NetResource) SetMycelium() (err error) {
	if nr.resource.Mycelium == nil {
		// no mycelium
		return nil
	}

	peers, err := environment.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get public mycelium peer list")
	}

	config := nr.resource.Mycelium
	// create the bridge.
	if err := nr.ensureMyceliumBridge(); err != nil {
		return err
	}

	keyFile := filepath.Join(nr.keyDir, nr.ID())

	defer func() {
		if err != nil {
			os.Remove(keyFile)
		}
	}()

	if err = os.WriteFile(keyFile, config.Key, 0444); err != nil {
		return errors.Wrap(err, "failed to store mycelium key")
	}

	if err := nr.ensureMyceliumNetwork(keyFile); err != nil {
		return err
	}

	name := nr.myceliumServiceName()

	init := zinit.Default()
	exists, err := init.Exists(name)
	if err != nil {
		return errors.Wrap(err, "failed to check mycelium service")
	}

	if exists {
		return nil
	}

	ns, err := nr.Namespace()
	if err != nil {
		return err
	}

	args := []string{
		"ip", "netns", "exec", ns,
		"mycelium",
		"--silent",
		"--key-file", keyFile,
		"--tun-name", "my",
		"--peers",
	}

	// first append peers from user input.
	// right now this is shadowed by Mycelium config validation
	// which does not allow custom peer list.
	args = AppendFunc(args, config.Peers, func(mp zos.MyceliumPeer) string {
		return string(mp)
	})

	// global peers list
	args = append(args, peers.Mycelium.Peers...)

	// todo: add custom peers requested by the user

	err = zinit.AddService(name, zinit.InitService{
		Exec: strings.Join(args, " "),
	})

	if err != nil {
		return errors.Wrap(err, "failed to add mycelium service for nr")
	}

	return init.Monitor(name)
}

func (nr *NetResource) ensureMyceliumNetwork(keyFile string) error {
	// applies network configuration for this mycelium instance inside the proper namespace
	mycelium, err := nr.inspectMycelium(keyFile)
	if err != nil {
		return err
	}

	subnet, gw, err := mycelium.Gateway()
	if err != nil {
		return err
	}

	nsName, err := nr.Namespace()
	if err != nil {
		return err
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return err
	}

	defer netNS.Close()

	bridgeName, err := nr.myceliumBridgeName()
	if err != nil {
		return err
	}

	if !ifaceutil.Exists(myceliumInterfaceName, netNS) {
		log.Debug().Str("create macvlan", myceliumInterfaceName).Msg("attach mycelium to bridge")
		if _, err := macvlan.Create(myceliumInterfaceName, bridgeName, netNS); err != nil {
			return err
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(myceliumInterfaceName)
		// this should not happen since it has been ensured before
		if err != nil {
			return err
		}
		// configure the bridge ip
		addresses, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return errors.Wrap(err, "failed to list my-br ip addresses")
		}

		if !slices.ContainsFunc(addresses, func(a netlink.Addr) bool {
			return slices.Equal(a.IP, gw.IP)
		}) {
			// If gw Ip is not configured, we set it up
			if err := netlink.AddrAdd(link, &netlink.Addr{
				IPNet: &gw,
			}); err != nil {
				return errors.Wrap(err, "failed to setup mycelium bridge address")
			}
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return errors.Wrap(err, "failed to bring mycelium macvtap up")
		}

		// also configure route to the subnet
		routes, err := netlink.RouteList(link, netlink.FAMILY_V6)
		if err != nil {
			return errors.Wrap(err, "failed to list mycelium routes")
		}
		if !slices.ContainsFunc(routes, func(r netlink.Route) bool {
			return r.Dst != nil && slices.Equal(r.Dst.IP, subnet.IP)
		}) {
			log.Debug().Str("gw", gw.IP.String()).Str("subnet", subnet.String()).Msg("adding mycelium route")

			if err := netlink.RouteAdd(&netlink.Route{
				Dst:       &subnet,
				LinkIndex: link.Attrs().Index,
			}); err != nil {
				return errors.Wrap(err, "failed to add mycelium route")
			}
		}

		return nil
	})

}

func (nr *NetResource) inspectMycelium(keyFile string) (inspection MyceliumInspection, err error) {
	_, err = os.Stat(keyFile)
	if err != nil {
		return inspection, err
	}

	// we check if the file exists before we do inspect because mycelium will create a random seed
	// file if file does not exist
	output, err := exec.Command("mycelium", "--key-file", keyFile, "inspect", "--json").Output()
	if err != nil {
		return inspection, errors.Wrap(err, "failed to inspect mycelium key")
	}

	if err := json.Unmarshal(output, &inspection); err != nil {
		return inspection, errors.Wrap(err, "failed to load mycelium information from key")
	}

	return inspection, nil
}

// only create the mycelium bridge inside the network resource.
// this is done anyway
func (nr *NetResource) ensureMyceliumBridge() error {
	name, err := nr.myceliumBridgeName()
	if err != nil {
		return err
	}

	if bridge.Exists(name) {
		return nil
	}

	log.Info().Str("bridge", name).Msg("create mycelium bridge")

	_, err = bridge.New(name)
	if err != nil {
		return err
	}

	if err := options.Set(name, options.IPv6Disable(true)); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}
	return nil
}

func wgIP(subnet *net.IPNet) *net.IPNet {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	return &net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(16, 32),
	}
}

// ConfigureWG sets the routes and IP addresses on the
// wireguard interface of the network resources
func (nr *NetResource) ConfigureWG(privateKey string) error {
	wgPeers, err := nr.wgPeers()
	if err != nil {
		return errors.Wrap(err, "failed to wireguard peer configuration")
	}

	nsName, err := nr.Namespace()
	if err != nil {
		return err
	}
	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return fmt.Errorf("network namespace %s does not exits", nsName)
	}

	handler := func(_ ns.NetNS) error {

		wgName, err := nr.WGName()
		if err != nil {
			return err
		}

		wg, err := wireguard.GetByName(wgName)
		if err != nil {
			return errors.Wrapf(err, "failed to get wireguard interface %s", wgName)
		}

		if err = wg.Configure(privateKey, int(nr.resource.WGListenPort), wgPeers); err != nil {
			return errors.Wrap(err, "failed to configure wireguard interface")
		}

		addrs, err := netlink.AddrList(wg, netlink.FAMILY_ALL)
		if err != nil {
			return err
		}
		curAddrs := mapset.NewSet()
		for _, addr := range addrs {
			curAddrs.Add(addr.IPNet.String())
		}

		newAddrs := mapset.NewSet()
		newAddrs.Add(wgIP(&nr.resource.Subnet.IPNet).String())

		toRemove := curAddrs.Difference(newAddrs)
		toAdd := newAddrs.Difference(curAddrs)

		log.Info().Msgf("current %s", curAddrs.String())
		log.Info().Msgf("to add %s", toAdd.String())
		log.Info().Msgf("to remove %s", toRemove.String())

		for addr := range toAdd.Iter() {
			addr, _ := addr.(string)
			log.Debug().Str("ip", addr).Msg("set ip on wireguard interface")
			if err := wg.SetAddr(addr); err != nil && !os.IsExist(err) {
				return errors.Wrapf(err, "failed to set address %s on wireguard interface %s", addr, wg.Attrs().Name)
			}
		}

		for addr := range toRemove.Iter() {
			addr, _ := addr.(string)
			log.Debug().Str("ip", addr).Msg("unset ip on wireguard interface")
			if err := wg.UnsetAddr(addr); err != nil && !os.IsNotExist(err) {
				return errors.Wrapf(err, "failed to unset address %s on wireguard interface %s", addr, wg.Attrs().Name)
			}
		}

		route := &netlink.Route{
			LinkIndex: wg.Attrs().Index,
			Dst:       &nr.networkIPRange,
		}
		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
			log.Error().
				Err(err).
				Str("route", route.String()).
				Msg("fail to set route")
			return errors.Wrapf(err, "failed to add route %s", route.String())
		}

		return nil
	}

	return netNS.Do(handler)
}

// Delete removes all the interfaces and namespaces created by the Create method
func (nr *NetResource) Delete() error {
	netnsName, err := nr.Namespace()
	if err != nil {
		return err
	}
	nrBrName, err := nr.BridgeName()
	if err != nil {
		return err
	}

	myBrName, err := nr.myceliumBridgeName()
	if err != nil {
		return err
	}

	myceliumName := nr.myceliumServiceName()
	init := zinit.Default()
	exists, err := init.Exists(myceliumName)
	if err == nil && exists {
		// we use StopMultiple instead of StopWait because multiple does an extra wait and
		// verification after a service is sig-killed
		if err := init.StopMultiple(10*time.Second, myceliumName); err != nil {
			log.Error().Err(err).Msg("failed to stop mycelium for network resource")
		}

		_ = init.Forget(myceliumName)
		_ = zinit.RemoveService(myceliumName)

		keyFile := filepath.Join(nr.keyDir, nr.ID())
		_ = os.Remove(keyFile)
	}

	if bridge.Exists(nrBrName) {
		if err := bridge.Delete(nrBrName); err != nil {
			log.Error().
				Err(err).
				Str("bridge", nrBrName).
				Msg("failed to delete network resource bridge")
			return err
		}
	}

	if bridge.Exists(myBrName) {
		if err := bridge.Delete(myBrName); err != nil {
			log.Error().
				Err(err).
				Str("bridge", myBrName).
				Msg("failed to delete network mycelium bridge")
			return err
		}
	}

	if namespace.Exists(netnsName) {
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
	}

	return nil
}

func (nr *NetResource) wgPeers() ([]*wireguard.Peer, error) {

	wgPeers := make([]*wireguard.Peer, 0, len(nr.resource.Peers)+1)

	for _, peer := range nr.resource.Peers {

		allowedIPs := make([]string, 0, len(peer.AllowedIPs))
		for _, ip := range peer.AllowedIPs {
			allowedIPs = append(allowedIPs, ip.String())
		}

		wgPeer := &wireguard.Peer{
			PublicKey:  string(peer.WGPublicKey),
			AllowedIPs: allowedIPs,
			Endpoint:   peer.Endpoint,
		}

		log.Info().Str("peer prefix", peer.Subnet.String()).Msg("generate wireguard configuration for peer")
		wgPeers = append(wgPeers, wgPeer)
	}

	return wgPeers, nil
}

func (nr *NetResource) createNetNS() error {
	name, err := nr.Namespace()
	if err != nil {
		return err
	}

	if namespace.Exists(name) {
		return nil
	}

	log.Info().Str("namespace", name).Msg("Create namespace")

	netNS, err := namespace.Create(name)
	if err != nil {
		return err
	}
	defer netNS.Close()
	err = netNS.Do(func(_ ns.NetNS) error {
		if err := options.SetIPv6Forwarding(true); err != nil {
			return err
		}
		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}
		return nil
	})
	return err
}

// attachToNRBridge creates a macvlan interface in the NR namespace, and attaches
// it to the NR bridge
func (nr *NetResource) attachToNRBridge() error {
	// are we really sure length is always <= 5 ?
	nsName, err := nr.Namespace()
	if err != nil {
		return err
	}
	nrIfaceName, err := nr.NRIface()
	if err != nil {
		return err
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return fmt.Errorf("network namespace %s does not exits", nsName)
	}
	defer netNS.Close()

	bridgeName, err := nr.BridgeName()
	if err != nil {
		return err
	}

	if !ifaceutil.Exists(nrIfaceName, netNS) {
		log.Debug().Str("create macvlan", nrIfaceName).Msg("attachNRToBridge")
		if _, err := macvlan.Create(nrIfaceName, bridgeName, netNS); err != nil {
			return err
		}
	}

	var handler = func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(nrIfaceName)
		if err != nil {
			return err
		}

		ipnet := nr.resource.Subnet
		ipnet.IP[len(ipnet.IP)-1] = 0x01
		log.Info().Str("addr", ipnet.String()).Msg("set address on macvlan interface")

		addr := &netlink.Addr{IPNet: &ipnet.IPNet, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil && !os.IsExist(err) {
			return err
		}

		ipv6 := Convert4to6(nr.ID(), ipnet.IP)
		addr = &netlink.Addr{IPNet: &net.IPNet{
			IP:   ipv6,
			Mask: net.CIDRMask(64, 128),
		}}
		if err = netlink.AddrAdd(link, addr); err != nil && !os.IsExist(err) {
			return err
		}

		addr = &netlink.Addr{IPNet: &net.IPNet{
			IP:   net.ParseIP("fe80::1"),
			Mask: net.CIDRMask(64, 128),
		}}
		if err = netlink.AddrAdd(link, addr); err != nil && !os.IsExist(err) {
			return err
		}

		return netlink.LinkSetUp(link)
	}
	return netNS.Do(handler)
}

// ensureNRBridge creates a bridge in the host namespace
// this bridge is used to connect containers from the name network
func (nr *NetResource) ensureNRBridge() error {
	name, err := nr.BridgeName()
	if err != nil {
		return err
	}

	if bridge.Exists(name) {
		return nil
	}

	log.Info().Str("bridge", name).Msg("Create bridge")

	_, err = bridge.New(name)
	if err != nil {
		return err
	}

	if err := options.Set(name, options.IPv6Disable(true)); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}
	return nil
}

// HasWireguard checks if network resource has wireguard setup up
func (nr *NetResource) HasWireguard() (bool, error) {
	nsName, err := nr.Namespace()
	if err != nil {
		return false, err
	}

	nrNetNS, err := namespace.GetByName(nsName)
	if err != nil {
		return false, err
	}

	defer nrNetNS.Close()

	wgName, err := nr.WGName()
	if err != nil {
		return false, err
	}
	exist := false
	err = nrNetNS.Do(func(_ ns.NetNS) error {
		_, err = wireguard.GetByName(wgName)

		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		} else if err != nil {
			return err
		}

		exist = true
		return nil
	})

	return exist, err
}

// SetWireguard sets wireguard of this network resource
func (nr *NetResource) SetWireguard(wg *wireguard.Wireguard) error {
	nsName, err := nr.Namespace()
	if err != nil {
		return err
	}

	nrNetNS, err := namespace.GetByName(nsName)
	if err != nil {
		return err
	}
	defer nrNetNS.Close()

	return netlink.LinkSetNsFd(wg, int(nrNetNS.Fd()))
}

func (nr *NetResource) applyFirewall() error {
	nsName, err := nr.Namespace()
	if err != nil {
		return err
	}

	buf := bytes.Buffer{}
	if err := fwTmpl.Execute(&buf, nil); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, nsName); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

// Convert4to6 converts a (private) ipv4 to the corresponding ipv6
func Convert4to6(netID string, ip net.IP) net.IP {
	h := md5.New()
	md5NetID := h.Sum([]byte(netID))

	// pick the last 2 bytes, handle ipv4 in both ipv6 form (leading 0 bytes)
	// and ipv4 form
	var lastbyte, secondtolastbyte byte
	if len(ip) == net.IPv6len {
		lastbyte = ip[15]
		secondtolastbyte = ip[14]
	} else if len(ip) == net.IPv4len {
		lastbyte = ip[3]
		secondtolastbyte = ip[2]
	}

	ipv6 := fmt.Sprintf("fd%x:%x%x:%x%x", md5NetID[0], md5NetID[1], md5NetID[2], md5NetID[3], md5NetID[4])
	ipv6 = fmt.Sprintf("%s:%x::%x", ipv6, secondtolastbyte, lastbyte)

	return net.ParseIP(ipv6)
}

// AppendFunc appends arrays with automatic map
func AppendFunc[A any, B any, S []A, D []B](d D, s S, f func(A) B) D {
	d = slices.Grow(d, len(s))

	for _, e := range s {
		d = append(d, f(e))
	}

	return d

}
