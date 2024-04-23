package pkg

import (
	"context"
	"fmt"
	"net"
	"reflect"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module network -version 0.0.1 -name network -package stubs github.com/threefoldtech/zos/pkg+Networker stubs/network_stub.go

// ContainerNetworkConfig defines how to construct the network namespace of a container
type ContainerNetworkConfig struct {
	IPs         []string
	PublicIP6   bool
	YggdrasilIP bool
}

// PlanetaryTap structure
type PlanetaryTap struct {
	Name    string
	HW      net.HardwareAddr
	IP      net.IPNet
	Gateway net.IPNet
}

type Interface struct {
	Name string
	IPs  []net.IPNet
	Mac  string
}

type Interfaces map[string]Interface

type ExitDevice struct {
	// IsSingle is set to true if br-pub
	// is connected to zos bridge
	IsSingle bool `json:"is_single"`
	// IsDual is set to true if br-pub is
	// connected to a physical nic
	IsDual bool `json:"is_dual"`
	// AsDualInterface is set to the physical
	// interface name if IsDual is true
	AsDualInterface string `json:"dual_interface"`
}

func (e *ExitDevice) String() string {
	if e.IsSingle {
		return "single"
	} else if e.IsDual {
		return fmt.Sprintf("dual(%s)", e.AsDualInterface)
	}

	return "unknown"
}

type NetResourceMetrics map[string]NetMetric

// Networker is the interface for the network module
type Networker interface {
	// Ready return nil is networkd is ready to operate
	// This function is used by other deamon to test if networkd is done booting
	Ready() error

	// Create a new network resource
	CreateNR(wl gridtypes.WorkloadID, network Network) (string, error)
	// Delete a network resource
	DeleteNR(wl gridtypes.WorkloadID) error

	// Namespace returns the namespace name for given netid.
	// it doesn't check if network exists.
	Namespace(id zos.NetID) string
	// deprecated all uses taps now

	// // Join a network (with network id) will create a new isolated namespace
	// // that is hooked to the network bridge with a veth pair, and assign it a
	// // new IP from the network resource range. The method return the new namespace
	// // name.
	// // The member name specifies the name of the member, and must be unique
	// // The NetID is the network id to join
	// Join(networkdID NetID, containerID string, cfg ContainerNetworkConfig) (join Member, err error)
	// // Leave delete a container nameapce created by Join
	// Leave(networkdID NetID, containerID string) (err error)

	// ZDBPrepare creates a network namespace with a macvlan interface into it
	// to allow the 0-db container to be publicly accessible
	// it retusn the name of the network namespace created
	// id is the zdb id (should be unique) is used to drive the hw mac
	// address for the interface so they always get the same IP
	ZDBPrepare(id string) (string, error)

	// ZDBDestroy is the opposite of ZDPrepare, it makes sure network setup done
	// for zdb is rewind. ns param is the namespace return by the ZDBPrepare
	ZDBDestroy(ns string) error

	// QSFSNamespace returns the namespace of the qsfs workload
	QSFSNamespace(id string) string

	// QSFSYggIP returns the ygg ip of the qsfs workload
	QSFSYggIP(id string) (string, error)

	// QSFSPrepare creates a network namespace with a macvlan interface into it
	// to allow qsfs container to reach the internet but not be reachable itself
	// it return the name of the network namespace created, and the ygg ip.
	// the id should be unique.
	QSFSPrepare(id string) (string, string, error)

	// QSFSDestroy rewind setup by QSFSPrepare
	QSFSDestroy(id string) error

	// SetupPrivTap sets up a tap device in the network namespace for the networkID. It is hooked
	// to the network bridge. The name of the tap interface is returned
	SetupPrivTap(networkID NetID, name string) (tap string, err error)

	// SetupMyceliumTap creates a new mycelium tap device attached to this network resource with deterministic IP address
	SetupMyceliumTap(name string, netID zos.NetID, config zos.MyceliumIP) (PlanetaryTap, error)

	// SetupYggTap sets up a tap device in the host namespace for the yggdrasil ip
	SetupYggTap(name string) (PlanetaryTap, error)

	// TapExists checks if the tap device with the given name exists already
	TapExists(name string) (bool, error)

	// RemoveTap removes the tap device with the given name
	RemoveTap(name string) error

	// PublicIPv4Support enabled on this node for reservations
	PublicIPv4Support() bool

	// SetupPubTap sets up a tap device in the host namespace for the public ip
	// reservation id. It is hooked to the public bridge. The name of the tap
	// interface is returned
	SetupPubTap(name string) (string, error)

	// PubTapExists checks if the tap device for the public network exists already
	PubTapExists(name string) (bool, error)

	// RemovePubTap removes the public tap device from the host namespace
	RemovePubTap(name string) error

	// SetupPubIPFilter sets up filter for this public ip
	SetupPubIPFilter(filterName string, iface string, ipv4 net.IP, ipv6 net.IP, mac string) error

	// RemovePubIPFilter removes the filter setted up by SetupPubIPFilter
	RemovePubIPFilter(filterName string) error

	// PubIPFilterExists checks if there is a filter installed with that name
	PubIPFilterExists(filterName string) bool
	// DisconnectPubTap disconnects the public tap from the network. The interface
	// itself is not removed and will need to be cleaned up later
	DisconnectPubTap(name string) error

	// GetSubnet of the network with the given ID on the local node
	GetSubnet(networkID NetID) (net.IPNet, error)

	// GetNet returns the full network range of the network
	GetNet(networkID NetID) (net.IPNet, error)

	// GetPublicIPv6Subnet returns the IPv6 prefix op the public subnet of the host
	GetPublicIPv6Subnet() (net.IPNet, error)

	// GetPublicIPV6Gateway gets the gateway for public IPv6 that can be used
	// by the VMs
	GetPublicIPV6Gateway() (net.IP, error)

	// GetDefaultGwIP returns the IPs of the default gateways inside the network
	// resource identified by the network ID on the local node, for IPv4 and IPv6
	// respectively
	GetDefaultGwIP(networkID NetID) (net.IP, net.IP, error)

	// GetIPv6From4 generates an IPv6 address from a given IPv4 address in a NR
	GetIPv6From4(networkID NetID, ip net.IP) (net.IPNet, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	// if iface is empty, return ALL interfaces in the given namespace
	// if they are physical
	Interfaces(iface string, netns string) (Interfaces, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	// [obsolete] please use Interfaces instead
	Addrs(iface string, netns string) (ips []net.IP, mac string, err error)

	WireguardPorts() ([]uint, error)

	// Public Config

	// Set node public namespace config.
	SetPublicConfig(cfg PublicConfig) error

	// UnsetPublicConfig removes public config from node
	UnsetPublicConfig() error

	// Get node public namespace config
	GetPublicConfig() (PublicConfig, error)

	// GetPublicExitDevice either return "singe" or "dual(<nic>)"
	GetPublicExitDevice() (ExitDevice, error)

	SetPublicExitDevice(iface string) error

	Metrics() (NetResourceMetrics, error)
	// Monitoring methods

	// ZOSAddresses monitoring streams for ZOS bridge IPs
	ZOSAddresses(ctx context.Context) <-chan NetlinkAddresses

	// DMZAddresses monitoring streams for dmz public interface
	DMZAddresses(ctx context.Context) <-chan NetlinkAddresses

	// YggAddresses monitoring streams for yggdrasil interface
	YggAddresses(ctx context.Context) <-chan NetlinkAddresses

	PublicAddresses(ctx context.Context) <-chan OptionPublicConfig
}

// Network type
type Network struct {
	zos.Network
	NetID NetID `json:"net_id"`
}

// NetID type
type NetID = zos.NetID

// IfaceType define the different public interface supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// PublicConfig is the configuration of the interface
// that is connected to the public internet
type PublicConfig struct {
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"type"`
	// Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 gridtypes.IPNet `json:"ipv4"`
	IPv6 gridtypes.IPNet `json:"ipv6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	// Domain is the node domain name like gent01.devnet.grid.tf
	// or similar
	Domain string `json:"domain"`
}

func (p *PublicConfig) IsEmpty() bool {
	return p.IPv4.Nil() && p.IPv6.Nil()
}

func PublicConfigFrom(cfg substrate.PublicConfig) (pub PublicConfig, err error) {
	pub.Type = MacVlanIface
	pub.IPv4, err = gridtypes.ParseIPNet(cfg.IP4.IP)
	if err != nil {
		return pub, err
	}
	pub.IPv6, err = gridtypes.ParseIPNet(cfg.IP6.AsValue.IP)
	if err != nil {
		return pub, err
	}
	pub.GW4 = net.ParseIP(cfg.IP4.GW)
	pub.GW6 = net.ParseIP(cfg.IP6.AsValue.GW)
	pub.Domain = cfg.Domain.AsValue

	return
}

func (p PublicConfig) Equal(cfg PublicConfig) bool {
	return reflect.DeepEqual(p, cfg)
}

type OptionPublicConfig struct {
	PublicConfig
	HasPublicConfig bool
}
