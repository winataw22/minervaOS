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
	Pause(ctx context.Context, twin uint32, id uint64) error
	Resume(ctx context.Context, twin uint32, id uint64) error
	Update(ctx context.Context, update gridtypes.Deployment) error
	Storage() Storage
	Twins() Twins
	Admins() Twins
}

// Provisioner interface. the errors returned by this interface are associated with
// provisioner errors, not workloads errors. The difference is, a failure to recognize the
// workload type for example, is a provisioner error. A workload error is when the workload
// fails to deploy and this is returned as Error state in the Result object (but nil error)
// Methods can return special error type ErrDidNotChange which instructs the engine that the
// workload provision was not carried on because it's already deployed, basically a no action
// needed indicator. In that case, the engine can ignore the returned result
type Provisioner interface {
	// Initialize is called before the provision engine is started
	Initialize(ctx context.Context) error
	// Provision a workload
	Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
	// Deprovision a workload
	Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error
	// Pause a workload
	Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
	// Resume a workload
	Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
	// Update a workload
	Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
	// CanUpdate checks if this workload can be updated on the fly
	CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool
}

// Filter is filtering function for Purge method

var (
	// ErrDeploymentExists returned if object exist
	ErrDeploymentExists = fmt.Errorf("exists")
	// ErrWorkloadExists returned if object exist
	ErrWorkloadExists = fmt.Errorf("exists")
	// ErrDeploymentConflict returned if deployment cannot be stored because
	// it conflicts with another deployment
	ErrDeploymentConflict = fmt.Errorf("conflict")
	//ErrDeploymentNotExists returned if object not exists
	ErrDeploymentNotExists = fmt.Errorf("deployment does not exist")
	// ErrWorkloadNotExist returned by storage if workload does not exist
	ErrWorkloadNotExist = fmt.Errorf("workload does not exist")
	// ErrNoActionNeeded can be returned by any provision method to indicate that
	// no action has been taken in case a workload is already deployed and the
	// engine then can skip updating the result of the workload.
	// When returned, the data returned by the provision is ignored
	ErrNoActionNeeded = fmt.Errorf("no action needed")
	// ErrDeploymentUpgradeValidationError error, is returned if the deployment
	// failed to compute upgrade steps
	ErrDeploymentUpgradeValidationError = fmt.Errorf("upgrade validation error")
	// ErrInvalidVersion invalid version error
	ErrInvalidVersion = fmt.Errorf("invalid version")
)

// Field interface
type Field interface{}
type VersionField struct {
	Version uint32
}

type DescriptionField struct {
	Description string
}

type MetadataField struct {
	Metadata string
}

type SignatureRequirementField struct {
	SignatureRequirement gridtypes.SignatureRequirement
}

type StorageCapacity struct {
	// Cap is total reserved capacity as per all active workloads
	Cap gridtypes.Capacity
	// Deployments is a list with all deployments that are active
	Deployments []gridtypes.Deployment
	// Workloads the total number of all workloads
	Workloads int
	// LastDeploymentTimestamp last deployment timestamp
	LastDeploymentTimestamp gridtypes.Timestamp
}

// Used with Storage interface to compute capacity, exclude any deployment
// and or workload that returns true from the capacity calculation.
type Exclude = func(dl *gridtypes.Deployment, wl *gridtypes.Workload) bool

// Storage interface
type Storage interface {
	// Create a new deployment in storage, it sets the initial transactions
	// for all workloads to "init" and the correct creation time.
	Create(deployment gridtypes.Deployment) error
	// Update updates a deployment fields
	Update(twin uint32, deployment uint64, fields ...Field) error
	// Delete deletes a deployment from storage.
	Delete(twin uint32, deployment uint64) error
	// Get gets the current state of a deployment from storage
	Get(twin uint32, deployment uint64) (gridtypes.Deployment, error)
	// Error sets global deployment error
	Error(twin uint32, deployment uint64, err error) error
	// Add workload to deployment, if no active deployment exists with same name
	Add(twin uint32, deployment uint64, workload gridtypes.Workload) error
	// Remove a workload from deployment.
	Remove(twin uint32, deployment uint64, name gridtypes.Name) error
	// Transaction append a transaction to deployment transactions logs
	Transaction(twin uint32, deployment uint64, workload gridtypes.Workload) error
	// Changes return all the historic transactions of a deployment
	Changes(twin uint32, deployment uint64) (changes []gridtypes.Workload, err error)
	// Current gets last state of a workload by name
	Current(twin uint32, deployment uint64, name gridtypes.Name) (gridtypes.Workload, error)
	// Twins list twins in storage
	Twins() ([]uint32, error)
	// ByTwin return list of deployments for a twin
	ByTwin(twin uint32) ([]uint64, error)
	// return total capacity and active deployments
	Capacity(exclude ...Exclude) (StorageCapacity, error)
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
