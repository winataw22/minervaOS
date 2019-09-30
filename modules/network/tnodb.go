package network

import (
	"errors"
	"net"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/types"
)

// TNoDB define the interface to implement
// to talk to a Tenant Network object database
type TNoDB interface {
	GetFarm(farm modules.Identifier) (*Farm, error)

	PublishInterfaces(node modules.Identifier, ifaces []types.IfaceInfo) error
	GetNode(modules.Identifier) (*types.Node, error)
	PublishWGPort(node modules.Identifier, ports []uint) error

	SetPublicIface(node modules.Identifier, pub *types.PubIface) error
	GetPubIface(node modules.Identifier) (*types.PubIface, error)
}

// TNoDBUtils define the interface to implement
// to talk to a Tenant Network object database including utils methods
type TNoDBUtils interface {
	TNoDB
	RegisterAllocation(farm modules.Identifier, allocation *net.IPNet) error
	RequestAllocation(farm modules.Identifier) (*net.IPNet, *net.IPNet, uint8, error)

	SelectExitNode(node modules.Identifier) error

	GetNetwork(netID modules.NetID) (*modules.Network, error)
	GetNetworksVersion(nodeID modules.Identifier) (versions map[modules.NetID]uint32, err error)
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
