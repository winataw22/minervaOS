package nr

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/options"

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

// NetResource holds the logic to configure an network resource
type NetResource struct {
	id pkg.NetID
	// local network resources
	resource pkg.Network
	// network IP range, usually a /16
	networkIPRange net.IPNet
}

// New creates a new NetResource object
// iprange is the full network subnet
func New(nr pkg.Network) (*NetResource, error) {
	return &NetResource{
		//fix here
		id:             nr.NetID,
		resource:       nr,
		networkIPRange: nr.NetworkIPRange.IPNet,
	}, nil
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

	if err := nr.createBridge(); err != nil {
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
	bridgeName, err := nr.BridgeName()
	if err != nil {
		return err
	}

	if bridge.Exists(bridgeName) {
		if err := bridge.Delete(bridgeName); err != nil {
			log.Error().
				Err(err).
				Str("bridge", bridgeName).
				Msg("failed to delete network resource bridge")
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

// createBridge creates a bridge in the host namespace
// this bridge is used to connect containers from the name network
func (nr *NetResource) createBridge() error {
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
