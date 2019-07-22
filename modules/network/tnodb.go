package network

import (
	"errors"
	"net"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/identity"
)

// TNoDB define the interface to implement
// to talk to a Tenant Network object database
type TNoDB interface {
	RegisterAllocation(farm identity.Identifier, allocation *net.IPNet) error
	RequestAllocation(farm identity.Identifier) (*net.IPNet, *net.IPNet, error)
	GetFarm(farm identity.Identifier) (Farm, error)

	PublishInterfaces() error

	ConfigurePublicIface(node identity.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error
	ReadPubIface(node identity.Identifier) (*PubIface, error)

	SelectExitNode(node identity.Identifier) error

	GetNetwork(netID modules.NetID) (*modules.Network, error)
	GetNetworksVersion(nodeID identity.Identifier) (versions map[modules.NetID]uint32, err error)
}

// Farm hold the ID, name and list of possible exit node of a farm
type Farm struct {
	ID        string   `json:"farm_id"`
	Name      string   `json:"name"`
	ExitNodes []string `json:"exit_nodes"`
}

// ErrNoPubIface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubIface = errors.New("no public interface configured for this node")
