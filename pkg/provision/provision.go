// Package provision is a daemon that pulls
// on reservation source, and then tries to
// apply these reservations locally.
// Note that, provision module doesn't expose
// any interface on zbus. since it should not
// be driven by users, instead all reservation
// should be pushed by the reservation source.
package provision

import (
	"context"

	"github.com/threefoldtech/zos/pkg"
)

// ReservationSource interface. The source
// defines how the node will get reservation requests
// then reservations are applied to the node to deploy
// a resource of the given Reservation.Type
type ReservationSource interface {
	Reservations(ctx context.Context) <-chan *Reservation
}

// Engine interface
type Engine interface {
	// Start the engine
	Run(ctx context.Context) error
	// Counters stream for number of provisioned resources
	Counters(ctx context.Context) <-chan pkg.ProvisionCounters
}

type provisioner func(ctx context.Context, reservation *Reservation) (interface{}, error)
type decommissioner func(ctx context.Context, reservation *Reservation) error

var (
	// provisioners defines the entry point for the different
	// reservation provisioners. Currently only containers are
	// supported.
	provisioners = map[ReservationType]provisioner{
		ContainerReservation:  containerProvision,
		VolumeReservation:     volumeProvision,
		NetworkReservation:    networkProvision,
		ZDBReservation:        zdbProvision,
		DebugReservation:      debugProvision,
		KubernetesReservation: kubernetesProvision,
	}

	decommissioners = map[ReservationType]decommissioner{
		ContainerReservation:  containerDecommission,
		VolumeReservation:     volumeDecommission,
		NetworkReservation:    networkDecommission,
		ZDBReservation:        zdbDecommission,
		DebugReservation:      debugDecommission,
		KubernetesReservation: kubernetesDecomission,
	}
)

// ErrTemporary is return when a reservation source failed to contact the BCDB
// user usually want to retry after getting this error
type ErrTemporary struct {
	err error
}

// NewErrTemporary wrap an error and mark it as temporary
func NewErrTemporary(err error) error {
	return ErrTemporary{err: err}
}

// Error implements the errors.Error interface
func (e ErrTemporary) Error() string {
	return e.err.Error()
}

// Is implements errors.Is interface
func (e ErrTemporary) Is(target error) bool {
	_, ok := target.(ErrTemporary)
	return ok
}

// Unwrap implements errors.Unwrap interface
func (e ErrTemporary) Unwrap() error {
	return e.err
}
