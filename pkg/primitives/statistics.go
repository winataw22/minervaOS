package primitives

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
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

type Reserved func() (gridtypes.Capacity, error)

// Statistics a provisioner interceptor that keeps track
// of consumed capacity. It also does validate of required
// capacity and then can report that this capacity can not be fulfilled
type Statistics struct {
	inner    provision.Provisioner
	total    gridtypes.Capacity
	reserved Reserved
	storage  provision.Storage
	mem      gridtypes.Unit
}

// NewStatistics creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatistics(total gridtypes.Capacity, storage provision.Storage, reserved Reserved, inner provision.Provisioner) *Statistics {
	vm, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	if reserved == nil {
		reserved = func() (gridtypes.Capacity, error) {
			return gridtypes.Capacity{}, nil
		}
	}

	return &Statistics{
		inner:    inner,
		total:    total,
		reserved: reserved,
		storage:  storage,
		mem:      gridtypes.Unit(vm.Total),
	}
}

type activeCounters struct {
	// used capacity from storage + reserved
	cap gridtypes.Capacity
	// Total deployments count
	deployments int
	// Total workloads count
	workloads int
}

// Get all used capacity from storage + reserved / deployments count and workloads count
func (s *Statistics) active() (activeCounters, error) {
	storageCap, err := s.storage.Capacity()
	if err != nil {
		return activeCounters{}, err
	}
	reserved, err := s.reserved()
	if err != nil {
		return activeCounters{}, err
	}
	storageCap.Cap.Add(&reserved)
	return activeCounters{
		storageCap.Cap,
		len(storageCap.Deployments),
		storageCap.Workloads,
	}, err
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

	activeCounters, err := s.active()
	cap := activeCounters.cap
	if err != nil {
		return cap, 0, err
	}

	m, err := mem.VirtualMemory()
	if err != nil {
		return cap, 0, err
	}

	theoreticalUsed := cap.MRU
	actualUsed := m.Total - m.Available
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

// UsersCounters the expected counters for deployments and workloads
type UsersCounters struct {
	// Total deployments count
	Deployments int `json:"deployments"`
	// Total workloads count
	Workloads int `json:"workloads"`
}

func (s *statisticsMessageBus) getCounters(ctx context.Context, payload []byte) (interface{}, error) {

	activeCounters, err := s.stats.active()
	if err != nil {
		return nil, err
	}

	reserved, err := s.stats.reserved()
	if err != nil {
		return nil, err
	}

	return struct {
		// Total system capacity
		Total gridtypes.Capacity `json:"total"`
		// Used capacity this include user + system resources
		Used gridtypes.Capacity `json:"used"`
		// System resource reserved by zos
		System gridtypes.Capacity `json:"system"`
		// Users statistics by zos
		Users UsersCounters `json:"users"`
	}{
		Total:  s.stats.Total(),
		Used:   activeCounters.cap,
		System: reserved,
		Users: UsersCounters{
			Deployments: activeCounters.deployments,
			Workloads:   activeCounters.workloads,
		},
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
				activeCounters, err := s.stats.active()
				if err != nil {
					log.Error().Err(err).Msg("failed to get used capacity")
				}
				ch <- activeCounters.cap
			}
		}
	}(ctx)
	return ch
}

func (s *statsStream) Current() (gridtypes.Capacity, error) {
	activeCounters, err := s.stats.active()
	return activeCounters.cap, err
}

func (s *statsStream) Total() gridtypes.Capacity {
	return s.stats.Total()
}
