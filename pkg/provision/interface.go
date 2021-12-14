package provision

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Twins is used to get twin public key
type Twins interface {
	GetKey(id uint32) ([]byte, error)
}

// Engine is engine interface
type Engine interface {
	// Provision pushes a workload to engine queue. on success
	// means that workload has been committed to storage (accepts)
	// and will be processes later
	Provision(ctx context.Context, wl gridtypes.Deployment) error
	Deprovision(ctx context.Context, twin uint32, id uint64, reason string) error
	Update(ctx context.Context, update gridtypes.Deployment) error
	Storage() Storage
	Twins() Twins
	Admins() Twins
}

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error)
	Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error
	Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error)
	CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool
}

// Filter is filtering function for Purge method

var (
	// ErrDeploymentExists returned if object exist
	ErrDeploymentExists = fmt.Errorf("exists")
	// ErrDeploymentConflict returned if deployment cannot be stored because
	// it conflicts with another deployment
	ErrDeploymentConflict = fmt.Errorf("conflict")
	//ErrDeploymentNotExists returned if object not exists
	ErrDeploymentNotExists = fmt.Errorf("not exists")
	// ErrDidNotChange special error that can be returned by the provisioner
	// if returned the engine does no update workload data
	ErrDidNotChange = fmt.Errorf("did not change")
	// ErrDeploymentUpgradeValidationError error, is returned if the deployment
	// failed to compute upgrade steps
	ErrDeploymentUpgradeValidationError = fmt.Errorf("upgrade validation error")
	// ErrInvalidVersion invalid version error
	ErrInvalidVersion = fmt.Errorf("invalid version")
)

// Storage interface
type Storage interface {
	Add(wl gridtypes.Deployment) error
	Set(wl gridtypes.Deployment) error
	Get(twin uint32, deployment uint64) (gridtypes.Deployment, error)
	Twins() ([]uint32, error)
	ByTwin(twin uint32) ([]uint64, error)

	// manage of shared workloads
	GetShared(twinID uint32, name gridtypes.Name) (gridtypes.WorkloadID, error)
	SharedByTwin(twinID uint32) ([]gridtypes.WorkloadID, error)
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
