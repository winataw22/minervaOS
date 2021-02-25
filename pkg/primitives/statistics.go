package primitives

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type (
	currentCapacityKey struct{}
)

// CurrentCapacity is a value that holds current system load in bytes
type CurrentCapacity struct {
	Cru uint64
	Mru uint64
	Hru uint64
	Sru uint64
}

// GetCapacity gets current capacity from context
func GetCapacity(ctx context.Context) CurrentCapacity {
	val := ctx.Value(currentCapacityKey{})
	if val == nil {
		panic("no current capacity injected")
	}

	return val.(CurrentCapacity)
}

// statsProvisioner a provisioner interceptor that keeps track
// of consumed capacity, and reprot it to the explorer
// when it has been changed
type statsProvisioner struct {
	inner    provision.Provisioner
	counters Counters
	reserved Counters
	mem      uint64

	nodeID string
}

// NewStatisticsProvisioner creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatisticsProvisioner(initial, reserved Counters, nodeID string, inner provision.Provisioner) provision.Provisioner {
	vm, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	return &statsProvisioner{inner: inner, counters: initial, reserved: reserved, nodeID: nodeID, mem: vm.Total}
}

func (s *statsProvisioner) currentCapacity() CurrentCapacity {
	return CurrentCapacity{
		Cru: s.counters.CRU.Current() + s.reserved.CRU.Current(),
		Mru: s.counters.CRU.Current() + s.reserved.MRU.Current(),
		Hru: s.counters.CRU.Current() + s.reserved.HRU.Current(),
		Sru: s.counters.CRU.Current() + s.reserved.SRU.Current(),
	}
}

func (s *statsProvisioner) hasEnoughCapacity(c *gridtypes.Capacity) error {
	//TODO: check required capacity here
	return nil
}

func (s *statsProvisioner) Provision(ctx context.Context, wl *gridtypes.Workload) (*gridtypes.Result, error) {
	current := s.currentCapacity()
	needed, err := wl.Capacity()
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate workload needed capacity")
	}

	if err := s.hasEnoughCapacity(&needed); err != nil {
		return nil, errors.Wrap(err, "failed to satisfy required capacity")
	}

	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	result, err := s.inner.Provision(ctx, wl)
	if err != nil {
		return result, err
	}

	if result.State == gridtypes.StateOk {
		if err := s.counters.Increment(wl); err != nil {
			log.Error().Err(err).Msg("failed to increment statistics counter")
		}
	}

	return result, nil
}

func (s *statsProvisioner) Decommission(ctx context.Context, wl *gridtypes.Workload) error {
	if err := s.inner.Decommission(ctx, wl); err != nil {
		return err
	}

	if err := s.counters.Decrement(wl); err != nil {
		log.Error().Err(err).Msg("failed to decrement statistics counter")
	}

	return nil
}

// func (s *statsProvisioner) shouldUpdateCounters(ctx context.Context, wl *gridtypes.Workload) (bool, error) {
// 	// rule, we always should update counters UNLESS it is a network reservation that
// 	// already have been counted before.
// 	if wl.Type != zos.NetworkType {
// 		return true, nil
// 	}

// 	var nr zos.Network
// 	if err := json.Unmarshal(wl.Data, &nr); err != nil {
// 		return false, fmt.Errorf("failed to unmarshal network from reservation: %w", err)
// 	}
// 	// otherwise we check the cache if a network
// 	// with the same id already exists
// 	id := zos.NetworkID(wl.User.String(), nr.Name)
// 	cache := provision.GetEngine(ctx).Storage()

// 	_, err := cache.GetNetwork(id)
// 	if errors.Is(err, provision.ErrWorkloadNotExists) {
// 		return true, nil
// 	} else if err != nil {
// 		return false, err
// 	}

// 	return false, nil
// }
