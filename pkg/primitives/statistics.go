package primitives

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type (
	currentCapacityKey struct{}
)

// GetCapacity gets current capacity from context
func GetCapacity(ctx context.Context) gridtypes.Capacity {
	val := ctx.Value(currentCapacityKey{})
	if val == nil {
		panic("no current capacity injected")
	}

	return val.(gridtypes.Capacity)
}

// Statistics a provisioner interceptor that keeps track
// of consumed capacity. It also does validate of required
// capacity and then can report that this capacity can not be fulfilled
type Statistics struct {
	inner    provision.Provisioner
	total    gridtypes.Capacity
	counters Counters
	reserved Counters
	mem      gridtypes.Unit
}

// round the given value to the lowest gigabyte
func roundTotalMemory(t gridtypes.Unit) gridtypes.Unit {
	return gridtypes.Unit(math.Floor(float64(t)/float64(gridtypes.Gigabyte))) * gridtypes.Gigabyte
}

// NewStatistics creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatistics(total, initial gridtypes.Capacity, reserved Counters, inner provision.Provisioner) *Statistics {
	vm, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	total.MRU = roundTotalMemory(total.MRU)

	log.Debug().Msgf("initial used capacity %+v", initial)
	var counters Counters
	counters.Increment(initial)

	return &Statistics{
		inner:    inner,
		total:    total,
		counters: counters,
		reserved: reserved,
		mem:      gridtypes.Unit(vm.Total),
	}
}

// Current returns the current used capacity
func (s *Statistics) Current() gridtypes.Capacity {
	total, usable, err := s.getUsableMemoryBytes()

	if err != nil {
		panic("failed to get memory consumption")
	}

	used := total + s.reserved.MRU.Current() - usable

	return gridtypes.Capacity{
		CRU:   uint64(s.counters.CRU.Current() + s.reserved.CRU.Current()),
		MRU:   used,
		HRU:   s.counters.HRU.Current() + s.reserved.HRU.Current(),
		SRU:   s.counters.SRU.Current() + s.reserved.SRU.Current(),
		IPV4U: uint64(s.counters.IPv4.Current()),
	}
}

// Total returns the node total capacity
func (s *Statistics) Total() gridtypes.Capacity {
	return s.total
}

// getUsableMemoryBytes returns the usable free memory. this takes
// into account the system reserved, actual available memory and the theorytical max reserved memory
// by the workloads
func (s *Statistics) getUsableMemoryBytes() (gridtypes.Unit, gridtypes.Unit, error) {
	m, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, err
	}

	reserved := s.reserved.MRU.Current()
	// total is always the total - the reserved
	total := s.total.MRU - reserved

	// which means the available memory is always also
	// is actual_available - reserved
	var available gridtypes.Unit
	if gridtypes.Unit(m.Available) > reserved {
		available = gridtypes.Unit(m.Available) - reserved
	}

	// get reserved memory from current deployed workloads (theoretical max)
	theoryticalUsed := s.counters.MRU.Current()
	var theoryticalFree gridtypes.Unit // this is the (total - workload)
	if total > theoryticalUsed {
		theoryticalFree = total - theoryticalUsed
	}

	// usable is the min of actual available on the system or theoryticalFree
	usable := gridtypes.Unit(math.Min(float64(theoryticalFree), float64(available)))
	return total, usable, nil
}

func (s *Statistics) hasEnoughCapacity(required *gridtypes.Capacity) error {
	// checks memory
	_, usable, err := s.getUsableMemoryBytes()
	if err != nil {
		return errors.Wrap(err, "failed to get available memory")
	}
	if required.MRU > usable {
		return fmt.Errorf("cannot fulfil required memory size %d bytes out of usable %d bytes", required.MRU, usable)
	}

	//check other resources as well?
	return nil
}

// Provision implements the provisioner interface
func (s *Statistics) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (result gridtypes.Result, err error) {
	current := s.Current()
	needed, err := wl.Capacity()
	if err != nil {
		return result, errors.Wrap(err, "failed to calculate workload needed capacity")
	}

	// we add extra overhead for some workload types here
	if wl.Type == zos.ZMachineType {
		// we add the min of 5% of allocated memory or 1G
		needed.MRU += gridtypes.Min(needed.MRU*5/100, gridtypes.Gigabyte)
	} // TODO: other types ?

	if err := s.hasEnoughCapacity(&needed); err != nil {
		return result, errors.Wrap(err, "failed to satisfy required capacity")
	}

	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	result, err = s.inner.Provision(ctx, wl)
	if err != nil {
		return result, err
	}

	if result.State == gridtypes.StateOk {
		log.Debug().Str("type", wl.Type.String()).Str("id", wl.ID.String()).Msgf("incrmenting capacity +%+v", needed)
		s.counters.Increment(needed)
	}

	return result, nil
}

// Decommission implements the decomission interface
func (s *Statistics) Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	if err := s.inner.Decommission(ctx, wl); err != nil {
		return err
	}
	cap, err := wl.Capacity()
	if err != nil {
		log.Error().Err(err).Msg("failed to decrement statistics counter")
		return nil
	}

	s.counters.Decrement(cap)

	return nil
}

// Update implements the provisioner interface
func (s *Statistics) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error) {
	return s.inner.Update(ctx, wl)
}

// CanUpdate implements the provisioner interface
func (s *Statistics) CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool {
	return s.inner.CanUpdate(ctx, typ)
}

// statistics api handlers for msgbus
type statisticsMessageBus struct {
	stats *Statistics
}

// NewStatisticsMessageBus register statistics handlers for message bus
func NewStatisticsMessageBus(router rmb.Router, stats *Statistics) error {
	api := statisticsMessageBus{stats}
	return api.setup(router)
}

func (s *statisticsMessageBus) setup(router rmb.Router) error {
	sub := router.Subroute("statistics")
	sub.WithHandler("get", s.getCounters)
	return nil
}

func (s *statisticsMessageBus) getCounters(ctx context.Context, payload []byte) (interface{}, error) {
	return struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}{
		Total: s.stats.Total(),
		Used:  s.stats.Current(),
	}, nil
}

type statsStream struct {
	stats *Statistics
}

func NewStatisticsStream(s *Statistics) pkg.Statistics {
	return &statsStream{s}
}

func (s *statsStream) Reserved(ctx context.Context) <-chan gridtypes.Capacity {
	ch := make(chan gridtypes.Capacity)
	go func(ctx context.Context) {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				ch <- s.stats.Current()
			}
		}
	}(ctx)
	return ch
}
