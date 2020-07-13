package provision

import (
	"context"

	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
)

// ReservationSource interface. The source
// defines how the node will get reservation requests
// then reservations are applied to the node to deploy
// a resource of the given Reservation.Type
type ReservationSource interface {
	Reservations(ctx context.Context) <-chan *Reservation
}

// ProvisionerFunc is the function called by the Engine to provision a workload
type ProvisionerFunc func(ctx context.Context, reservation *Reservation) (interface{}, error)

// DecomissionerFunc is the function called by the Engine to decomission a workload
type DecomissionerFunc func(ctx context.Context, reservation *Reservation) error

// ReservationConverterFunc is used to convert from the explorer workloads type into the
// internal Reservation type
type ReservationConverterFunc func(w workloads.Workloader) (*Reservation, error)

//ResultConverterFunc is used to convert internal Result type to the explorer workload result
type ResultConverterFunc func(result Result) (*workloads.Result, error)

// ReservationCache define the interface to store
// some reservations
type ReservationCache interface {
	Add(r *Reservation) error
	Get(id string) (*Reservation, error)
	Remove(id string) error
	Exists(id string) (bool, error)
	Sync(Statser) error
}

// Feedbacker defines the method that needs to be implemented
// to send the provision result to BCDB
type Feedbacker interface {
	Feedback(nodeID string, r *Result) error
	Deleted(nodeID, id string) error
	UpdateStats(nodeID string, w directory.WorkloadAmount, u directory.ResourceAmount) error
}

// Signer interface is used to sign reservation result before
// sending them to the explorer
type Signer interface {
	Sign(b []byte) ([]byte, error)
}

// Statser is used by the provision Engine to keep
// track of how much resource unit and number of primitives
// is provisionned
type Statser interface {
	Increment(r *Reservation) error
	Decrement(r *Reservation) error
	CurrentUnits() directory.ResourceAmount
	CurrentWorkloads() directory.WorkloadAmount
}
