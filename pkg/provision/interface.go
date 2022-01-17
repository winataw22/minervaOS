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

// Provisioner interface. the errors returned by this interface are associated with
// provisioner errors, not workloads errors. The difference is, a failure to recognize the
// workload type for example, is a provisioner error. A workload error is when the workload
// fails to deploy and this is returned as Error state in the Result object (but nil error)
// Methods can return special error type ErrDidNotChange which instructs the engine that the
// workload provision was not carried on because it's already deployed, basically a no action
// needed indicator. In that case, the engine can ignore the returned result
type Provisioner interface {
	Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
	Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error
	Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
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
	ErrDeploymentNotExists = fmt.Errorf("workload does not exist")
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

// ErrUnchanged can be returned by the Provisioner.Update it means
// that the update has failed but the workload is intact
type ErrUnchanged struct {
	cause error
}

// NewUnchangedError return an instance of ErrUnchanged
func NewUnchangedError(cause error) error {
	if cause == nil {
		return nil
	}

	return ErrUnchanged{cause}
}

func (e ErrUnchanged) Unwrap() error {
	return e.cause
}

func (e ErrUnchanged) Error() string {
	return e.cause.Error()
}

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
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
