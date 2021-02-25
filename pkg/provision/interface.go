package provision

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Users is used to get user public key
type Users interface {
	GetKey(id gridtypes.ID) (ed25519.PublicKey, error)
}

// Engine is engine interface
type Engine interface {
	// Provision pushes a workload to engine queue. on success
	// means that workload has been committed to storage (accepts)
	// and will be processes later
	Provision(ctx context.Context, wl gridtypes.Workload) error
	Deprovision(ctx context.Context, id gridtypes.ID, reason string) error
	Storage() Storage
	Users() Users
	Admins() Users
}

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, wl *gridtypes.Workload) (*gridtypes.Result, error)
	Decommission(ctx context.Context, wl *gridtypes.Workload) error
}

// Filter is filtering function for Purge method

var (
	//ErrWorkloadExists returned if object exist
	ErrWorkloadExists = fmt.Errorf("exists")
	//ErrWorkloadNotExists returned if object not exists
	ErrWorkloadNotExists = fmt.Errorf("not exists")
)

// Storage interface
type Storage interface {
	Add(wl gridtypes.Workload) error
	Set(wl gridtypes.Workload) error
	Get(id gridtypes.ID) (gridtypes.Workload, error)
	GetNetwork(id zos.NetID) (gridtypes.Workload, error)

	ByType(t gridtypes.WorkloadType) ([]gridtypes.ID, error)
	ByUser(user gridtypes.ID, t gridtypes.WorkloadType) ([]gridtypes.ID, error)
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
