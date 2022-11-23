package primitives

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
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

var (
	_ provision.Provisioner = (*Statistics)(nil)
)

// Statistics a provisioner interceptor that keeps track
// of consumed capacity. It also does validate of required
// capacity and then can report that this capacity can not be fulfilled
type Statistics struct {
	inner    provision.Provisioner
	total    gridtypes.Capacity
	reserved gridtypes.Capacity
	storage  provision.Storage
	mem      gridtypes.Unit
}

// NewStatistics creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatistics(total gridtypes.Capacity, storage provision.Storage, reserved gridtypes.Capacity, inner provision.Provisioner) *Statistics {
	vm, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	return &Statistics{
		inner:    inner,
		total:    total,
		reserved: reserved,
		storage:  storage,
		mem:      gridtypes.Unit(vm.Total),
	}
}

// Get all used capacity from storage + reserved
func (s *Statistics) active() (gridtypes.Capacity, error) {
	cap, _, err := s.storage.Capacity()
	cap.Add(&s.reserved)
	return cap, err
}

// Current returns the current used capacity including reserved capacity
// used by the system
func (s *Statistics) Current() (gridtypes.Capacity, error) {
	return s.active()
}

// Total returns the node total capacity
func (s *Statistics) Total() gridtypes.Capacity {
	return s.total
}

// getUsableMemoryBytes returns the used capacity by *reservations* and usable free memory. for the memory
// it takes into account reserved memory for the system
func (s *Statistics) getUsableMemoryBytes() (gridtypes.Capacity, gridtypes.Unit, error) {
	// [                          ]
	// [[R][ WL ]                 ]
	// [[    actual    ]          ]

	cap, err := s.active()
	if err != nil {
		return cap, 0, err
	}

	m, err := mem.VirtualMemory()
	if err != nil {
		return cap, 0, err
	}

	theoreticalUsed := cap.MRU
	actualUsed := (m.Total - m.Available) + uint64(s.reserved.MRU)

	used := gridtypes.Max(theoreticalUsed, gridtypes.Unit(actualUsed))

	usable := gridtypes.Unit(m.Total) - used
	return cap, usable, nil
}

func (s *Statistics) hasEnoughCapacity(required *gridtypes.Capacity) (gridtypes.Capacity, error) {
	// checks memory
	used, usable, err := s.getUsableMemoryBytes()
	if err != nil {
		return used, errors.Wrap(err, "failed to get available memory")
	}
	if required.MRU > usable {
		return used, fmt.Errorf("cannot fulfil required memory size %d bytes out of usable %d bytes", required.MRU, usable)
	}

	//check other resources as well?
	return used, nil
}

// Initialize implements provisioner interface
func (s *Statistics) Initialize(ctx context.Context) error {
	return s.inner.Initialize(ctx)
}

// Provision implements the provisioner interface
func (s *Statistics) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (result gridtypes.Result, err error) {
	needed, err := wl.Capacity()
	if err != nil {
		return result, errors.Wrap(err, "failed to calculate workload needed capacity")
	}

	current, err := s.hasEnoughCapacity(&needed)
	if err != nil {
		return result, errors.Wrap(err, "failed to satisfy required capacity")
	}

	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	return s.inner.Provision(ctx, wl)
}

// Decommission implements the decomission interface
func (s *Statistics) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	return s.inner.Deprovision(ctx, wl)
}

// Update implements the provisioner interface
func (s *Statistics) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error) {
	return s.inner.Update(ctx, wl)
}

// CanUpdate implements the provisioner interface
func (s *Statistics) CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool {
	return s.inner.CanUpdate(ctx, typ)
}

func (s *Statistics) Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error) {
	return s.inner.Pause(ctx, wl)
}

func (s *Statistics) Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error) {
	return s.inner.Resume(ctx, wl)
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

	used, err := s.stats.Current()
	if err != nil {
		return nil, err
	}
	return struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}{
		Total: s.stats.Total(),
		Used:  used,
	}, nil
}

type statsStream struct {
	stats *Statistics
}

func NewStatisticsStream(s *Statistics) pkg.Statistics {
	return &statsStream{s}
}

func (s *statsStream) ReservedStream(ctx context.Context) <-chan gridtypes.Capacity {
	ch := make(chan gridtypes.Capacity)
	go func(ctx context.Context) {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Minute):
				used, err := s.stats.Current()
				if err != nil {
					log.Error().Err(err).Msg("failed to get used capacity")
				}
				ch <- used
			}
		}
	}(ctx)
	return ch
}

func (s *statsStream) Current() (gridtypes.Capacity, error) {
	return s.stats.Current()
}

func (s *statsStream) Total() gridtypes.Capacity {
	return s.stats.Total()
}
