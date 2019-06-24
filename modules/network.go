package modules

import (
	"net"

	"github.com/vishvananda/netlink"
)

//go:generate mkdir -p stubs

//go:generate zbusc -module network -version 0.0.1 -name network -package stubs github.com/threefoldtech/zosv2/modules+Networker stubs/network_stub.go

//Networker is the interface for the network module
type Networker interface {
	GetNetwork(id string) (*Network, error)
	ApplyNetResource(*Network) error
	DeleteNetResource(*Network) error
}

// NetID is a type defining the ID of a network
type NetID string

// ReachabilityV4 is the Node's IPv4 reachability:
type ReachabilityV4 int

const (
	// ReachabilityV4Hidden The Node lives in an RFC1918 space, can't listen publically
	ReachabilityV4Hidden ReachabilityV4 = iota
	// ReachabilityV4Public The Node's Wireguard interfaces listen address is reachable publicly
	ReachabilityV4Public
)

// ReachabilityV6 is the Node's IPv6 reachability
type ReachabilityV6 int

const (
	// ReachabilityV6ULA The Node lives in an ULA prefix (IPv6 private space)
	ReachabilityV6ULA ReachabilityV6 = iota
	// ReachabilityV6Public The Node's Wireguard interfaces listen address is reachable publicly
	ReachabilityV6Public
)

// NodeID is a type defining a node ID
type NodeID struct {
	ID string
	// FarmeerID is needed for when a Node is HIDDEN, but lives in the same farm.
	// that way if a network resource is started on a HIDDEN Node, and the peer
	// is also HIDDEN, but part of the same farm, we can surmise that that peer
	// can be included for that network resource
	// https://www.wireguard.com/protocol/ -> we could send a handshake request
	// to a HIDDEN peer and in case we receive a reply, include the peer in the list
	FarmerID       string
	ReachabilityV4 ReachabilityV4
	ReachabilityV6 ReachabilityV6
}

// Network represent a full network owned by a user
type Network struct {
	// some type of identification... an uuid ?
	// that netid is bound to a user and an allowed (bought) creation of a
	// node-local prefix for a bridge/container/vm
	// needs to be queried from somewhere(TBD) to be filled in
	NetID NetID
	// a netresource is a group of interconnected prefixes for a netid
	// needs to be queried and updated when the netresource is created
	Resources []*NetResource
	// the exit is the ultimate default gateway container
	// as well the prefix as the local config needs to be queried.
	// - the prefix from the grid
	// - the exit prefix and default gw from the local allocation
	Exit *ExitPoint
	// AllocationNr is for when a new allocation has been necessary and needs to
	// be added to the pool for Prefix allocations.
	// this is needed as we set up deterministic interface names, that could conflict with
	// the already existing allocation-derived names
	AllocationNR int8
}

// NetResource represent a part of a network configuration
type NetResource struct {
	// where does it live
	NodeID NodeID
	// prefix is the IPv6 allocation that will be connected to the
	// bridge/container/vm
	Prefix *net.IPNet
	// Gateways in IPv6 are link-local. To be able to use IPv6 in any way,
	// an interface needs an IPv6 link-local address. As wireguard interfaces
	// are l3-only, the kernel doesn't assign one, so we need to assign one
	// ourselves. (we need to come up with a deterministic way, so we can be
	// sure we now which/where)
	LinkLocal *net.IPNet
	// what are the peers:
	// each netresource needs to know what prefixes are reachable through
	// what endpoint. Basically this `peers` array will be used to build
	// up the wireguard config in each netresource.
	Peers []*Peer
	// a list of firewall rules to open access directly, IF that netresource
	// would be directly routed (future)
	// IPv6Allow []net.IPNet
}

// ExitPoint represents the exit container(ns) hold as well a prefix as netresource as well
// an IPv6 address that is going to hold the routes for all prefixes of the
// network. That IPv6 address will thus be the gateway for all prefixes that
// are part of that network. That also means that an upstream router needs to
// know the list of prefixes that need to be routed to that IPv6 address.
// An upstream router is the entry point toward nodes that have only IPv6 access
// through tunnels (like nodes in ipv4-only networks or home networks)
type ExitPoint struct {
	// netresource is the same as on all other netresources of a tenant network
	*NetResource
	// the ultimate IPv{4,6} config of the exit container.
	ipv4Conf ipv4Conf
	ipv4DNAT []DNAT

	ipv6Conf  ipv6Conf
	ipv6Allow []net.IP
}

// DNAT represents an ipv4/6 portforwarding/firewalling
type DNAT struct {
	InternalIP   net.IP
	InternalPort int16

	ExternalIP   net.IP
	ExternalPort int16
}

type ipv4Conf struct {
	// cidr
	CIDR    net.IPNet
	Gateway net.IP
	Metric  uint32
	// deterministic name in function of the prefix and it's allocation
	Iface netlink.Link
	// TBD, we need to establish if we want fc00/7 (ULA) or rfc1918 networks
	// to be NATed (6to4 and/or 66)
	EnableNAT bool
}

// comments: see above
type ipv6Conf struct {
	Adder   net.IP
	Gateway net.IP
	metric  uint32
	Iface   netlink.Link
}

// Peer is a peer for which we have a tunnel established and the
// prefix it routes to. The connection, as it is a peer to peer connection,
// can be of type wireguard, but also any other type that can bring
// a packet to a node containing a netresource.
// If for instance that node lives in the same subnet, it'll be a lot more
// efficient to set up a vxlan (multicast or with direct fdb entries), than
// using wireguard tunnels (that can be seen in a later phase)
type Peer struct {
	Type       ConnType
	Prefix     *net.IPNet
	Connection Wireguard
}

// ConnType is an enum
type ConnType int

const (
	// ConnTypeWireguard is an ConnType enum value for wireguard
	ConnTypeWireguard ConnType = iota
	// ConnTypeLocalVxlan
)

// Wireguard represent a wireguard interface configuration
// the key would be a public key, with the private key only available
// locally and stored locally.
type Wireguard struct {
	// TBD, a peer can be IPv6, IPv6-ll or IPv4
	IP net.IP
	// Listen port of wireguard
	Port uint16
	// base64 encoded public key
	// Key []byte
	Key string
}

// definition for later usage
// an l2vxlan wil be connected to a default bridge that gets attached to the
// network resource. That way we can easily add a vxlan to that bridge for
// local interconnectivity
// type l2vxlan {
//     // deterministic or stored...
//     NICName string
//     // Or it's through fdb entries
//     Option<Vec<peer>>
//     // Or it's in a multicast vxlan
//     Option<group>
//     // a vxlan always has an ID
//     id
// }
