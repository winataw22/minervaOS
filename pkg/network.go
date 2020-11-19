package pkg

import (
	"context"
	"fmt"
	"net"

	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/versioned"
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
	CreateNR(NetResource) (string, error)
	// Delete a network resource
	DeleteNR(NetResource) error

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
	// hw is an optional hardware address that will be set on the new interface
	ZDBPrepare(hw net.HardwareAddr) (string, error)

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

	// GetSubnet of the network with the given ID on the local node
	GetSubnet(networkID NetID) (net.IPNet, error)

	// GetDefaultGwIP returns the IP(v4) of the default gateway inside the network
	// resource identified by the network ID on the local node
	GetDefaultGwIP(networkID NetID) (net.IP, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	Addrs(iface string, netns string) ([]net.IP, error)

	// ZOSAddresses monitoring streams for ZOS bridge IPs
	ZOSAddresses(ctx context.Context) <-chan NetlinkAddresses

	// DMZAddresses monitoring streams for dmz public interface
	DMZAddresses(ctx context.Context) <-chan NetlinkAddresses

	// YggAddresses monitoring streams for yggdrasil interface
	YggAddresses(ctx context.Context) <-chan NetlinkAddresses

	PublicAddresses(ctx context.Context) <-chan NetlinkAddresses
}

// Network represent the description if a user private network
type Network struct {
	Name string `json:"name"`
	//unique id inside the reservation is an autoincrement (USE AS NET_ID)
	NetID NetID `json:"net_id"`
	// IP range of the network, must be an IPv4 /16
	IPRange types.IPNet `json:"ip_range"`

	NetResources []NetResource `json:"net_resources"`
}

// NetResource is the description of a part of a network local to a specific node
type NetResource struct {
	Name string
	//unique id inside the reservation is an autoincrement (USE AS NET_ID)
	NetID NetID `json:"net_id"`
	// IP range of the network, must be an IPv4 /16
	NetworkIPRange types.IPNet `json:"ip_range"`

	NodeID string `json:"node_id"`
	// IPV4 subnet for this network resource
	Subnet types.IPNet `json:"subnet"`

	WGPrivateKey string `json:"wg_private_key"`
	WGPublicKey  string `json:"wg_public_key"`
	WGListenPort uint16 `json:"wg_listen_port"`

	Peers []Peer `json:"peers"`
}

// Valid checks if the network resource is valid.
func (nr *NetResource) Valid() error {
	if nr.NetID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if nr.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	if nr.NetworkIPRange.Nil() {
		return fmt.Errorf("network IP range cannot be empty")
	}

	if nr.NodeID == "" {
		return fmt.Errorf("network resource node ID cannot empty")
	}
	if len(nr.Subnet.IP) == 0 {
		return fmt.Errorf("network resource subnet cannot empty")
	}

	if nr.WGPrivateKey == "" {
		return fmt.Errorf("network resource wireguard private key cannot empty")
	}

	if nr.WGPublicKey == "" {
		return fmt.Errorf("network resource wireguard public key cannot empty")
	}

	if nr.WGListenPort == 0 {
		return fmt.Errorf("network resource wireguard listen port cannot empty")
	}

	for _, peer := range nr.Peers {
		if err := peer.Valid(); err != nil {
			return err
		}
	}

	return nil
}

// Peer is the description of a peer of a NetResource
type Peer struct {
	// IPV4 subnet of the network resource of the peer
	Subnet types.IPNet `json:"subnet"`

	WGPublicKey string        `json:"wg_public_key"`
	AllowedIPs  []types.IPNet `json:"allowed_ips"`
	Endpoint    string        `json:"endpoint"`
}

// NetID is a type defining the ID of a network
type NetID string

// Version of the network workloads keep in cache by networkd
var (
	NetworkSchemaV1 = versioned.MustParse("1.0.0")
	NetworkSchemaV2 = versioned.MustParse("2.0.0")
	// NetworkSchemaLatestVersion network object latest version
	NetworkSchemaLatestVersion = NetworkSchemaV2
)

// Valid checks if peer is valid
func (p *Peer) Valid() error {
	if p.WGPublicKey == "" {
		return fmt.Errorf("peer wireguard public key cannot empty")
	}

	if p.Subnet.Nil() {
		return fmt.Errorf("peer wireguard subnet cannot empty")
	}

	if len(p.AllowedIPs) <= 0 {
		return fmt.Errorf("peer wireguard allowedIPs cannot empty")
	}
	return nil
}
