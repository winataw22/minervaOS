package pkg

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module network -version 0.0.1 -name network -package stubs github.com/threefoldtech/zos/pkg+Networker stubs/network_stub.go

// Member holds information about a the network namespace of a container
type Member struct {
	Namespace   string
	IPv6        net.IP
	IPv4        net.IP
	YggdrasilIP net.IP
}

// ContainerNetworkConfig defines how to construct the network namespace of a container
type ContainerNetworkConfig struct {
	IPs         []string
	PublicIP6   bool
	YggdrasilIP bool
}

//Networker is the interface for the network module
type Networker interface {
	// Ready return nil is networkd is ready to operate
	// This function is used by other deamon to test if networkd is done booting
	Ready() error

	// Create a new network resource
	CreateNR(Network) (string, error)
	// Delete a network resource
	DeleteNR(Network) error

	// Join a network (with network id) will create a new isolated namespace
	// that is hooked to the network bridge with a veth pair, and assign it a
	// new IP from the network resource range. The method return the new namespace
	// name.
	// The member name specifies the name of the member, and must be unique
	// The NetID is the network id to join
	Join(networkdID NetID, containerID string, cfg ContainerNetworkConfig) (join Member, err error)
	// Leave delete a container nameapce created by Join
	Leave(networkdID NetID, containerID string) (err error)

	// ZDBPrepare creates a network namespace with a macvlan interface into it
	// to allow the 0-db container to be publicly accessible
	// it retusn the name of the network namespace created
	// id is the zdb id (should be unique) is used to drive the hw mac
	// address for the interface so they always get the same IP
	ZDBPrepare(id string) (string, error)

	// ZDBDestroy is the opposite of ZDPrepare, it makes sure network setup done
	// for zdb is rewind. ns param is the namespace return by the ZDBPrepare
	ZDBDestroy(ns string) error

	// SetupTap sets up a tap device in the network namespace for the networkID. It is hooked
	// to the network bridge. The name of the tap interface is returned
	SetupTap(networkID NetID) (string, error)

	// TapExists checks if the tap device exists already
	TapExists(networkID NetID) (bool, error)

	// RemoveTap removes the tap device from the network namespace
	// of the networkID
	RemoveTap(networkID NetID) error

	// PublicIPv4Support enabled on this node for reservations
	PublicIPv4Support() bool

	// SetupPubTap sets up a tap device in the host namespace for the public ip
	// reservation id. It is hooked to the public bridge. The name of the tap
	// interface is returned
	SetupPubTap(PubIPReservationID string) (string, error)

	// PubTapExists checks if the tap device for the public network exists already
	PubTapExists(PubIPReservationID string) (bool, error)

	// RemovePubTap removes the public tap device from the host namespace
	RemovePubTap(PubIPReservationID string) error

	// DisconnectPubTap disconnects the public tap from the network. The interface
	// itself is not removed and will need to be cleaned up later
	DisconnectPubTap(PubIPReservationID string) error

	// GetSubnet of the network with the given ID on the local node
	GetSubnet(networkID NetID) (net.IPNet, error)

	// GetNet returns the full network range of the network
	GetNet(networkID NetID) (net.IPNet, error)

	// GetPublicIPv6Subnet returns the IPv6 prefix op the public subnet of the host
	GetPublicIPv6Subnet() (net.IPNet, error)

	// GetDefaultGwIP returns the IPs of the default gateways inside the network
	// resource identified by the network ID on the local node, for IPv4 and IPv6
	// respectively
	GetDefaultGwIP(networkID NetID) (net.IP, net.IP, error)

	// GetIPv6From4 generates an IPv6 address from a given IPv4 address in a NR
	GetIPv6From4(networkID NetID, ip net.IP) (net.IPNet, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	Addrs(iface string, netns string) ([]net.IP, error)

	WireguardPorts() ([]uint, error)

	// Set node public namespace config
	SetPublicConfig(cfg PublicConfig) error

	// Get node public namespace config
	GetPublicConfig() (PublicConfig, error)

	// ZOSAddresses monitoring streams for ZOS bridge IPs
	ZOSAddresses(ctx context.Context) <-chan NetlinkAddresses

	// DMZAddresses monitoring streams for dmz public interface
	DMZAddresses(ctx context.Context) <-chan NetlinkAddresses

	// YggAddresses monitoring streams for yggdrasil interface
	YggAddresses(ctx context.Context) <-chan NetlinkAddresses

	PublicAddresses(ctx context.Context) <-chan NetlinkAddresses
}

// Network type
type Network struct {
	zos.Network
	NetID             NetID  `json:"net_id"`
	WGPrivateKeyPlain string `json:"wireguard_private_key"`
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

	// Version int `json:"version"`
}
