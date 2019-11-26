package types

import (
	"fmt"
	"net"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/schema"
)

// IfaceType define the different public interface supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// IfaceInfo is the information about network interfaces
// that the node will publish publicly
// this is used to be able to configure public side of a node
type IfaceInfo struct {
	Name    string   `json:"name"`
	Addrs   []IPNet  `json:"addrs"`
	Gateway []net.IP `json:"gateway"`
}

// DefaultIP return the IP address of the interface that has a default gateway configured
// this function currently only check IPv6 addresses
func (i *IfaceInfo) DefaultIP() (net.IP, error) {
	if len(i.Gateway) <= 0 {
		return nil, fmt.Errorf("interface has not gateway")
	}

	for _, addr := range i.Addrs {
		if addr.IP.IsLinkLocalUnicast() ||
			addr.IP.IsLinkLocalMulticast() ||
			addr.IP.To4() != nil {
			continue
		}

		if addr.IP.To16() != nil {
			return addr.IP, nil
		}
	}
	return nil, fmt.Errorf("no ipv6 address with default gateway")
}

// PubIface is the configuration of the interface
// that is connected to the public internet
type PubIface struct {
	Master string `json:"master"`
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"type"`
	Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 IPNet `json:"ipv4"`
	IPv6 IPNet `json:"ipv6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	Version int `json:"version"`
}

// Node is the public information about a node
type Node struct {
	NodeID string `json:"node_id"`
	FarmID uint64 `json:"farm_id"`

	Ifaces []*IfaceInfo `json:"ifaces"`

	PublicConfig *PubIface `json:"public_config"`
	ExitNode     int       `json:"exit_node"`
	WGPorts      []uint    `json:"wg_ports"`
}

// NewNodeFromSchema converts a TfgridNode2 into Node
func NewNodeFromSchema(node directory.TfgridNode2) *Node {
	n := &Node{
		NodeID: node.NodeID,
		FarmID: node.FarmID,

		Ifaces: make([]*IfaceInfo, len(node.Ifaces)),

		PublicConfig: nil,
		ExitNode:     0,
		WGPorts:      node.WGPorts,
	}
	if node.Ifaces != nil {
		for i, iface := range node.Ifaces {
			n.Ifaces[i] = &IfaceInfo{
				Name:    iface.Name,
				Addrs:   make([]IPNet, len(iface.Addrs)),
				Gateway: iface.Gateway,
			}

			for y, addr := range iface.Addrs {
				n.Ifaces[i].Addrs[y] = NewIPNetFromSchema(addr)
			}
		}
	}
	if node.PublicConfig != nil {
		n.PublicConfig = &PubIface{
			Master: node.PublicConfig.Master,
			Type:   IfaceType(node.PublicConfig.Type.String()),
			// Vlan: node.PublicConfig.
			IPv4: NewIPNetFromSchema(node.PublicConfig.Ipv4),
			IPv6: NewIPNetFromSchema(node.PublicConfig.Ipv6),

			GW4: node.PublicConfig.Gw4,
			GW6: node.PublicConfig.Gw6,

			Version: int(node.PublicConfig.Version),
		}
	}

	return n
}

// IPNet type
type IPNet struct{ net.IPNet }

// NewIPNet creates a new IPNet from net.IPNet
func NewIPNet(n *net.IPNet) IPNet {
	return IPNet{IPNet: *n}
}

// NewIPNetFromSchema creates an IPNet from schema.IPRange
func NewIPNetFromSchema(n schema.IPRange) IPNet {
	return IPNet{n.IPNet}
}

// ParseIPNet parse iprange
func ParseIPNet(txt string) (r IPNet, err error) {
	if len(txt) == 0 {
		//empty ip net value
		return r, nil
	}
	//fmt.Println("parsing: ", string(text))
	ip, net, err := net.ParseCIDR(txt)
	if err != nil {
		return r, err
	}

	net.IP = ip
	r.IPNet = *net
	return
}

// MustParseIPNet prases iprange, panics if invalid
func MustParseIPNet(txt string) IPNet {
	r, err := ParseIPNet(txt)
	if err != nil {
		panic(err)
	}
	return r
}

// UnmarshalText loads IPRange from string
func (i *IPNet) UnmarshalText(text []byte) error {
	v, err := ParseIPNet(string(text))
	if err != nil {
		return err
	}

	i.IPNet = v.IPNet
	return nil
}

// MarshalJSON dumps iprange as a string
func (i IPNet) MarshalJSON() ([]byte, error) {
	if len(i.IPNet.IP) == 0 {
		return []byte(`""`), nil
	}
	v := fmt.Sprint("\"", i.String(), "\"")
	return []byte(v), nil
}

func (i IPNet) String() string {
	return i.IPNet.String()
}

// Nil returns true if IPNet is not set
func (i *IPNet) Nil() bool {
	return i.IP == nil && i.Mask == nil
}

// ToSchema creates a schema IPRange from IPNet
func (i *IPNet) ToSchema() schema.IPRange {
	return schema.IPRange{IPNet: i.IPNet}
}
