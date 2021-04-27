package primitives

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
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
	mem      uint64

	nodeID string
}

// NewStatistics creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatistics(total, initial gridtypes.Capacity, reserved Counters, nodeID string, inner provision.Provisioner) *Statistics {
	vm, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	log.Debug().Msgf("initial used capacity %+v", initial)
	var counters Counters
	counters.Increment(initial)
	ram := math.Ceil(float64(vm.Total) / (1024 * 1024 * 1024))
	return &Statistics{
		inner:    inner,
		total:    total,
		counters: counters,
		reserved: reserved,
		mem:      uint64(ram),
		nodeID:   nodeID,
	}
}

// Current returns the current used capacity
func (s *Statistics) Current() gridtypes.Capacity {
	return gridtypes.Capacity{
		CRU:   s.counters.CRU.Current() + s.reserved.CRU.Current(),
		MRU:   s.counters.MRU.Current() + s.reserved.MRU.Current(),
		HRU:   s.counters.HRU.Current() + s.reserved.HRU.Current(),
		SRU:   s.counters.SRU.Current() + s.reserved.SRU.Current(),
		IPV4U: s.counters.IPv4.Current(),
	}
}

// Total returns the node total capacity
func (s *Statistics) Total() gridtypes.Capacity {
	return s.total
}

func (s *Statistics) hasEnoughCapacity(used *gridtypes.Capacity, required *gridtypes.Capacity) error {
	// checks memory
	if required.MRU+used.MRU > s.mem {
		return fmt.Errorf("cannot fulfil required memory size")
	}

	//check other as well?

	return nil
}

// Provision implements the provisioner interface
func (s *Statistics) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error) {
	current := s.Current()
	needed, err := wl.Capacity()
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate workload needed capacity")
	}

	if err := s.hasEnoughCapacity(&current, &needed); err != nil {
		return nil, errors.Wrap(err, "failed to satisfy required capacity")
	}

	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	result, err := s.inner.Provision(ctx, wl)
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
func (s *Statistics) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error) {
	return s.inner.Update(ctx, wl)
}

// CanUpdate implements the provisioner interface
func (s *Statistics) CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool {
	return s.inner.CanUpdate(ctx, typ)
}

type statisticsAPI struct {
	stats *Statistics
}

// NewStatisticsAPI sets up a new statistics API and set it up on the given
// router
func NewStatisticsAPI(router *mux.Router, stats *Statistics) error {
	api := statisticsAPI{stats}
	return api.setup(router)
}

func (s *statisticsAPI) setup(router *mux.Router) error {
	router.Path("/counters").HandlerFunc(mw.AsHandlerFunc(s.getCounters)).Methods(http.MethodGet).Name("statistics-counters")
	return nil
}

func (s *statisticsAPI) getCounters(r *http.Request) (interface{}, mw.Response) {
	return struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}{
		Total: s.stats.Total(),
		Used:  s.stats.Current(),
	}, nil
}
