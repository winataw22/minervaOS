package ip

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/vishvananda/netlink"
)

// Nibble is an helper struct used to generate
// deterministic name based on an IPv6 address
type Nibble struct {
	allocNr int8
	nibble  []byte
}

// NewNibble create a new Nibble object
func NewNibble(prefix *net.IPNet, allocNr int8) (*Nibble, error) {

	if prefix == nil || prefix.IP == nil || prefix.Mask == nil {
		return nil, fmt.Errorf("prefix cannot be nil")
	}
	size, _ := prefix.Mask.Size()
	if size != 64 {
		return nil, fmt.Errorf("allocation prefix can only be a /64")
	}
	if allocNr < 0 {
		return nil, fmt.Errorf("allocNr cannot be negative")
	}

	return &Nibble{
		nibble:  []byte(prefix.IP)[6:8],
		allocNr: allocNr,
	}, nil
}

// Hex return the hexadecimal version of the meaningfull nibble
func (n *Nibble) Hex() string {
	return fmt.Sprintf("%x", n.nibble)
}

// WGName return the deterministic wireguard name in the Network Resource
func (n *Nibble) WGName() string {
	return fmt.Sprintf("wg-%s-%d", n.Hex(), n.allocNr)
}

// WireguardPort return the deterministic wireguard listen port
func (n *Nibble) WireguardPort() uint16 {
	return binary.BigEndian.Uint16(n.nibble)
}

// BridgeName return the deterministic bridge name of the Network Resource
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.Hex(), n.allocNr)
}

// NamespaceName return the deterministic Namespace name
func (n *Nibble) NamespaceName() string {
	return fmt.Sprintf("net-%s-%d", n.Hex(), n.allocNr)
}

// NRLocalName return the deterministic veth interface name
// added here for compliance to docs
func (n *Nibble) NRLocalName() string {
	return fmt.Sprintf("veth-%s-%d", n.Hex(), n.allocNr)
}

// EPPubName return the deterministic public interface name
// this Interface points to the veth peer GWtoEPName
func (n *Nibble) EPPubName() string {
	return fmt.Sprintf("pub-%s-%d", n.Hex(), n.allocNr)
}

// EP4PubName return the deterministic public interface name
// this Interface points to the ipv4 bridge
func (n *Nibble) EP4PubName() string {
	return fmt.Sprintf("ip4-%s-%d", n.Hex(), n.allocNr)
}

// Br4PubName return the deterministic interface name
// of the veth pair attahed to the gateway ipv4 bridge
func (n *Nibble) Br4PubName() string {
	return fmt.Sprintf("br4-%s-%d", n.Hex(), n.allocNr)
}

// EPPubLL ExitPoint Public Link-Local
// the interface that faces the other side of the veth into the GW
// we differentiate it by shifting 2 bytes, having 0001 in the last 2
func (n *Nibble) EPPubLL() *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::%s:1", n.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
}

// EPPubIP4R ExitPoint Pub routing IPv4 on lo
func (n *Nibble) EPPubIP4R() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 1, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(16, 32),
	}
}

// NRLocalIP4 returns the IPv4 address of a network resource
func (n *Nibble) NRLocalIP4() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, n.nibble[0], n.nibble[1], 1),
		Mask: net.CIDRMask(24, 32),
	}
}

// WGAllowedIP4 returns the IPv4 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedIP4() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 255, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(32, 32),
	}
}

// WGAllowedIP4Net returns the IPv4 network resource subnet address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedIP4Net() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, n.nibble[0], n.nibble[1], 0),
		Mask: net.CIDRMask(24, 32),
	}
}

// WGAllowedIP6 returns the IPv6 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedIP6() *net.IPNet {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	b[14] = n.nibble[0]
	b[15] = n.nibble[1]

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(128, 128),
	}
}

// WGLL returns the fe80 address to be used in wireguard link local
func (n *Nibble) WGLL() *net.IPNet {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	b[14] = n.nibble[0]
	b[15] = n.nibble[1]

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(64, 128),
	}
}

// WGIP4 returns the IPv4 of the wireguard interface
func (n *Nibble) WGIP4() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 1, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(16, 32),
	}
}

// WGIP4RT returns the routing IPv4 of the wireguard interface
func (n *Nibble) WGIP4RT() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 255, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(16, 32),
	}
}

// RouteIPv6Exit returns the route to the exit point
func (n *Nibble) RouteIPv6Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
	}
}

// WGExitPeerAllowIPs returns the list of allowed IPs of a wireguard interface
func WGExitPeerAllowIPs() []*net.IPNet {
	output := make([]*net.IPNet, 2)
	output[0] = &net.IPNet{
		IP:   net.ParseIP("0.0.0.0"),
		Mask: net.CIDRMask(0, 32),
	}
	output[1] = &net.IPNet{
		IP:   net.ParseIP("::"),
		Mask: net.CIDRMask(0, 128),
	}
	return output
}

// WGEndpoint returns the value for the endpoint configuration of a wireguard interface
func WGEndpoint(peer *modules.Peer) string {
	var endpoint string
	if peer.Connection.IP.To16() != nil {
		endpoint = fmt.Sprintf("[%s]:%d", peer.Connection.IP.String(), peer.Connection.Port)
	} else {
		endpoint = fmt.Sprintf("%s:%d", peer.Connection.IP.String(), peer.Connection.Port)
	}
	return endpoint
}

// // WGLL returns the fe80 address to be used in wireguard allowed ip configuration
// func (n *Nibble) WGLL() net.IP {
// 	return net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex()))
// }

//GWDefaultRoute (to be renamed) is the gateway in an NR for ::
//that is: the route to the ExitPoint
func (n *Nibble) GWDefaultRoute() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
	}
}

// RouteIPv4Exit (to be renamed) adds the route for another NR
func (n *Nibble) RouteIPv4Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("10.%d.%d.0", n.nibble[0], n.nibble[1])),
			Mask: net.CIDRMask(24, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

// RouteIPv4DefaultExit (to be renamed) is the gateway in an NR for ::
// that is: the route to the ExitPoint
func (n *Nibble) RouteIPv4DefaultExit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

// RouteIPv4DefaultContainer is the gateway in an NR for ::
// that is: the route to the Network rearouse IP
func (n *Nibble) RouteIPv4DefaultContainer() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.%d.%d.1", n.nibble[0], n.nibble[1])),
	}
}

// EPToGWName return the deterministic nic name of the EXitPoint NR to the gateway
func (n *Nibble) EPToGWName() string {
	return fmt.Sprintf("to-%s-%d", n.Hex(), n.allocNr)
}

// GWPubName is the name of the iface facing the penultimate router
// format pub-X-Y
// where X =  exitnode nr, Y = allocationnr
func GWPubName(exitnodenr, allocnr int) string {
	return fmt.Sprintf("pub-%d-%d", exitnodenr, allocnr)
}

// GWPubIP6 is the IP on the prefixzero segment of the GW container (BAR)
// it returns the list of IP that need to be installed on
// GWPubName() interface.
// that is :
//  - prefix:ExitNodeNr()::1
func GWPubIP6(prefix net.IP, exitnodenr int) *net.IPNet {
	b := make([]byte, net.IPv6len)
	copy(b, prefix[:6])
	binary.BigEndian.PutUint16(b[8:], uint16(exitnodenr)<<12)
	b[net.IPv6len-1] = 0x001
	return &net.IPNet{IP: net.IP(b), Mask: net.CIDRMask(64, 128)}
}

// GWPubLL is an added link-local address of the iface facing the router
// Format fe80::X000:0:0:1/64 and
//     $prefix:X000:0:0:1/64
// where X = exitnodenr
func GWPubLL(exitnodenr int) *net.IPNet {
	b := make([]byte, net.IPv6len)
	binary.BigEndian.PutUint16(b[:2], 0xfe80)
	binary.BigEndian.PutUint16(b[8:], uint16(uint16(exitnodenr)<<12))
	binary.BigEndian.PutUint16(b[14:], 0x001)

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(64, 128),
	}
}

// GWtoEPName is the gw Container interface facing the Exitpoint veth peer
func (n *Nibble) GWtoEPName() string {
	return fmt.Sprintf("to-%s-%d", n.Hex(), n.allocNr)
}

// GWtoEPLL is the link-local address on the iface in the wg facing the
// Exitpoint's pub iface (veth pair)
func (n *Nibble) GWtoEPLL() *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::1:%s", n.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
}

// NRDefaultRoute returns the default route of a exit point
func (n *Nibble) NRDefaultRoute() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: n.GWtoEPLL().IP,
	}
}

// ExitNodeRange TODO
func ExitNodeRange(prefix *net.IPNet, exitnodenr uint8, rnd uint16) *net.IPNet {
	b := make([]byte, net.IPv6len)
	copy(b, prefix.IP)
	b[6] = ((exitnodenr) << 4) | uint8(((rnd >> 8) & 0x0f))
	b[7] = byte(rnd & 0x00ff)

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(64, 128),
	}
}
