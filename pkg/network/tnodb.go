package network

import (
	"errors"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// TNoDB define the interface to implement
// to talk to a Tenant Network object database
type TNoDB interface {
	GetFarm(farm pkg.Identifier) (*Farm, error)

	PublishInterfaces(node pkg.Identifier, ifaces []types.IfaceInfo) error
	GetNode(pkg.Identifier) (*types.Node, error)
	PublishWGPort(node pkg.Identifier, ports []uint) error

	SetPublicIface(node pkg.Identifier, pub *types.PubIface) error
	GetPubIface(node pkg.Identifier) (*types.PubIface, error)
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
