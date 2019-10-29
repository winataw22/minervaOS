# 0-OS v2 Provisioning schemas

Generic reservation type
```go
// ReservationType type
type ReservationType string

const (
	// ContainerReservation type
	ContainerReservation ReservationType = "container"
	// VolumeReservation type
	VolumeReservation ReservationType = "volume"
	// NetworkReservation type
	NetworkReservation ReservationType = "network"
)

// ReplyTo defines how report the result of the provisioning operation
type ReplyTo string

// Tenant defines the tenant identity
type Tenant string

func (t Tenant) String() string {
	return string(t)
}

// Reservation struct
type Reservation struct {
	// ID of the reservation
	ID string `json:"id"`
	// Tenant ID
	Tenant Tenant `json:"tenant"`
	// ReplyTo is a dummy attribute to hold the 3bot address
	// we need to report to once the reservation is done
	ReplyTo ReplyTo `json:"reply-to"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type ReservationType `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data"`
}
```

Container provision schema

```go
// Network struct
type Network struct {
	NetworkID string `json:"network_id"`
}
```

```
@url = tfgrid.reservation.network
network_id = (S)
```

```go
// Mount defines a container volume mounted inside the container
type Mount struct {
	VolumeID   string `json:"volume_id"`
	Mountpoint string `json:"mountpoint"`
}

//Container creation info
type Container struct {
	// URL of the flist
	FList string `json:"flist"`
	// Env env variables to container in format
	Env map[string]string `json:"env"`
	// Entrypoint the process to start inside the container
	Entrypoint string `json:"entrypoint"`
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool `json:"interactive"`
	// Mounts extra mounts in the container
	Mounts []Mount `json:"mounts"`
	// Network network info for container
	Network Network `json:"network"`
}
```

```
@url = tfgrid.reservation.container.mount
volume_id = (S)
mountpoint = (S)

@url = tfgrid.reservation.container
flist = (S)
environment = (dict)
entrypoint = (S)
interactive = true (B)
volumes = (LO) !tfgrid.reservation.container.mount
network = (O) !tfgrid.reservation.network
```

Volume provision schema

```go
// DiskType defines disk type
type DiskType string

const (
	// HDDDiskType for hdd disks
	HDDDiskType DiskType = "HDD"
	// SSDDiskType for ssd disks
	SSDDiskType DiskType = "SSD"
)

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type DiskType `json:"type"`
}
```

```
@url = tfgrid.reservation.volume
id = (S)
size = (I)
type = "HDD,SSD" (E)
```

Network provision schema

```go
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
	ID string `json:"id"`
	// FarmeerID is needed for when a Node is HIDDEN, but lives in the same farm.
	// that way if a network resource is started on a HIDDEN Node, and the peer
	// is also HIDDEN, but part of the same farm, we can surmise that that peer
	// can be included for that network resource
	// https://www.wireguard.com/protocol/ -> we could send a handshake request
	// to a HIDDEN peer and in case we receive a reply, include the peer in the list
	FarmerID       string         `json:"farmer_id"`
	ReachabilityV4 ReachabilityV4 `json:"reachability_v4"`
	ReachabilityV6 ReachabilityV6 `json:"reachability_v6"`
}


// Network represent a full network owned by a user
type Network struct {
	// some type of identification... an uuid ?
	// that netid is bound to a user and an allowed (bought) creation of a
	// node-local prefix for a bridge/container/vm
	// needs to be queried from somewhere(TBD) to be filled in
	NetID NetID `json:"network_id"`

	PrefixZero *net.IPNet

	// a netresource is a group of interconnected prefixes for a netid
	// needs to be queried and updated when the netresource is created
	Resources []*NetResource `json:"resources"`
	// the exit is the ultimate default gateway container
	// as well the prefix as the local config needs to be queried.
	// - the prefix from the grid
	// - the exit prefix and default gw from the local allocation
	Exit *ExitPoint `json:"exit_point"`
	// AllocationNr is for when a new allocation has been necessary and needs to
	// be added to the pool for Prefix allocations.
	// this is needed as we set up deterministic interface names, that could conflict with
	// the already existing allocation-derived names
	AllocationNR int8 `json:"allocation_nr"`

	// Version is an incremental number updated each time the network object
	// is changed. This allow node to know when a network object needs to re-applied
	Version uint32 `json:"version"`
}

// NetResource represent a part of a network configuration
type NetResource struct {
	// where does it live
	NodeID *NodeID `json:"node_id"`
	// prefix is the IPv6 allocation that will be connected to the
	// bridge/container/vm
	Prefix *net.IPNet `json:"prefix"`
	// Gateways in IPv6 are link-local. To be able to use IPv6 in any way,
	// an interface needs an IPv6 link-local address. As wireguard interfaces
	// are l3-only, the kernel doesn't assign one, so we need to assign one
	// ourselves. (we need to come up with a deterministic way, so we can be
	// sure we now which/where)
	LinkLocal *net.IPNet `json:"link_local"`
	// what are the peers:
	// each netresource needs to know what prefixes are reachable through
	// what endpoint. Basically this `peers` array will be used to build
	// up the wireguard config in each netresource.
	Peers []*Peer `json:"peers"`
	// a list of firewall rules to open access directly, IF that netresource
	// would be directly routed (future)
	// IPv6Allow []net.IPNet

	// Mark this NetResource as the exit point of the network
	ExitPoint bool `json:"exit_point"`
}

// Peer is a peer for which we have a tunnel established and the
// prefix it routes to. The connection, as it is a peer to peer connection,
// can be of type wireguard, but also any other type that can bring
// a packet to a node containing a netresource.
// If for instance that node lives in the same subnet, it'll be a lot more
// efficient to set up a vxlan (multicast or with direct fdb entries), than
// using wireguard tunnels (that can be seen in a later phase)
type Peer struct {
	Type       ConnType   `json:"type"`
	Prefix     *net.IPNet `json:"prefix"`
	Connection Wireguard  `json:"connection"`
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
	IP net.IP `json:"ip"`
	// Listen port of wireguard
	Port uint16 `json:"port"`
	// base64 encoded public key
	// Key []byte
	Key        string `json:"key"`
	PrivateKey string `json:"private_key"`
}

// ExitPoint represents the exit container(ns) hold as well a prefix as netresource as well
// an IPv6 address that is going to hold the routes for all prefixes of the
// network. That IPv6 address will thus be the gateway for all prefixes that
// are part of that network. That also means that an upstream router needs to
// know the list of prefixes that need to be routed to that IPv6 address.
// An upstream router is the entry point toward nodes that have only IPv6 access
// through tunnels (like nodes in ipv4-only networks or home networks)
type ExitPoint struct {
	// the ultimate IPv{4,6} config of the exit container.
	Ipv4Conf *Ipv4Conf `json:"ipv4_conf"`
	Ipv4DNAT []*DNAT   `json:"ipv4_dnat"`

	Ipv6Conf  *Ipv6Conf `json:"ipv6_conf"`
	Ipv6Allow []net.IP  `json:"ipv6_allow"`
}

// DNAT represents an ipv4/6 portforwarding/firewalling
type DNAT struct {
	InternalIP   net.IP `json:"internal_ip"`
	InternalPort uint16 `json:"internal_port"`

	ExternalIP   net.IP `json:"external_ip"`
	ExternalPort uint16 `json:"external_port"`

	Protocol string `json:"protocol"`
}

//Ipv4Conf represents the the IPv4 configuration of an exit container
type Ipv4Conf struct {
	// cidr
	CIDR    *net.IPNet `json:"cird"`
	Gateway net.IP     `json:"gateway"`
	Metric  uint32     `json:"metric"`
	// deterministic name in function of the prefix and it's allocation
	Iface string `json:"iface"`
	// TBD, we need to establish if we want fc00/7 (ULA) or rfc1918 networks
	// to be NATed (6to4 and/or 66)
	EnableNAT bool `json:"enable_nat"`
}

//Ipv6Conf represents the the IPv6 configuration of an exit container
type Ipv6Conf struct {
	Addr    *net.IPNet `json:"addr"`
	Gateway net.IP     `json:"gateway"`
	Metric  uint32     `json:"metric"`
	Iface   string     `json:"iface"`
}
```


```
@url = tfgrid.reservation.network
net_id = (S)
prefix_zero = (iprange)
resources = (LO) !tfgrid.reservation.network.net_resource
exit_point = (O) !tfgrid.reservation.network.exit_point
allocation_nr = (I)
version = (I)

@url = tfgrid.reservation.network.node_id
id = (S)
farm_id = (S)
reachability_v4 = "PUBLIC,HIDDEN" (E)
reachability_v6 = "PUBLIC,HIDDEN" (E)

@url = tfgrid.reservation.network.net_resource
node_id = (O) !tfgrid.reservation.network.node_id
prefix = (iprange)
link_local = (iprange)
peers = (LO) ! tfgrid.reservation.network.peer
exit_point = (B)

@url = tfgrid.reservation.network.peer
prefix = (iprange)
connection = (O) !tfgrid.reservation.network.wireguard

@url = tfgrid.reservation.network.wireguard
ip = (ipaddr)
port = (ipport)
key = (S)
private_key = (S)

@url = tfgrid.reservation.network.exit_point
ip4_conf = (O) !tfgrid.reservation.network.ip4_conf
ip4_dnat = (O) !tfgrid.reservation.network.ip4_dnat

ip6_conf = (O) !tfgrid.reservation.network.ip6_conf
ip6_allow = (Lipaddr)

@url = tfgrid.reservation.network.ip4_conf
cidr = (iprange)
gateway = (ipaddr)
metric = (I)
iface = (S)
enable_nat = (B)

@url = !tfgrid.reservation.network.ip6_conf
addr = (iprange)
gateway = (ipaddr)
metric = (I)
iface = (S)
```

Farm network allocation

```go
type allocation struct {
	Allocation *net.IPNet
	SubNetUsed []uint64
}
```

```
@url = !tfgrid.reservation.network.allocation
allocation = (iprange)
sub_net_used = (LI)
```

Farm info
```go
type Farm struct {
	ID        string   `json:"farm_id"`
	Name      string   `json:"name"`
	ExitNodes []string `json:"exit_nodes"`
}
```

```
@url = !tfgrid.farm
id = (S)
name = (S)
exit_nodes = (LS)
```

Node detail

```go

// IfaceInfo is the information about network interfaces
// that the node will publish publicly
// this is used to be able to configure public side of a node
type IfaceInfo struct {
    Name    string       `json:"name"`
	Addrs   []*net.IPNet `json:"addrs"`
	Gateway []net.IP     `json:"gateway"`
}
```

```
@url = !tfgrid.node.iface
name = (S)
addrs = (Liprange)
gateway = (Lipaddr)
```

```go
// IfaceType define the different public interface supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// PubIface is the configuration of the interface
// that is connected to the public internet
type PubIface struct {
	Master string `json:"master"`
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"iface_type"`
	Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 *net.IPNet `json:"ip_v4"`
	IPv6 *net.IPNet `json:"ip_v6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	Version int `json:"version"`
}
```

```
@url = tfgrid.node.public_iface
master = (S)
type = "macvlan" (E)
ipv4 = (ipaddr)
ipv6 = (ipaddr)
gw4 = (ipaddr)
gw6 = (ipaddr)
version = (I)
```

```go
// Node is the public information about a node
type Node struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`

	Ifaces []*IfaceInfo `json:"ifaces"`

	PublicConfig *PubIface `json:"public_config"`
	ExitNode     bool      `json:"exit_node"`
}
```

```
@url = tfgrid.node
node_id = (S)
fam_id = (S)
ifaces = (LO) !tfgrid.node.iface
public_config = (O)!tfgrid.node.public_iface
exit_node = (B)
```