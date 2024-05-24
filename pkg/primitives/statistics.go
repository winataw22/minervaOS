package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/kernel"
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
	// last deployment timestamp
	lastDeploymentTimestamp gridtypes.Timestamp
}

// Get all used capacity from storage + reserved / deployments count and workloads count
func (s *Statistics) active(exclude ...provision.Exclude) (activeCounters, error) {
	storageCap, err := s.storage.Capacity(exclude...)
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
		storageCap.LastDeploymentTimestamp,
	}, err
}

// Total returns the node total capacity
func (s *Statistics) Total() gridtypes.Capacity {
	return s.total
}

// getUsableMemoryBytes returns the used capacity by *reservations* and usable free memory. for the memory
// it takes into account reserved memory for the system
// excluding (not including it as 'used' any workload or deployment that matches the exclusion list)
func (s *Statistics) getUsableMemoryBytes(exclude ...provision.Exclude) (gridtypes.Capacity, gridtypes.Unit, error) {
	// [                          ]
	// [[R][ WL ]                 ]
	// [[    actual    ]          ]

	activeCounters, err := s.active(exclude...)
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

func (s *Statistics) hasEnoughCapacity(wl *gridtypes.WorkloadWithID) (gridtypes.Capacity, error) {
	required, err := wl.Capacity()
	if err != nil {
		return gridtypes.Capacity{}, errors.Wrap(err, "failed to calculate workload needed capacity")
	}

	// get used capacity by ALL workloads excluding this workload
	// we do that by providing an exclusion list
	used, usable, err := s.getUsableMemoryBytes(func(dl_ *gridtypes.Deployment, wl_ *gridtypes.Workload) bool {
		id, _ := gridtypes.NewWorkloadID(dl_.TwinID, dl_.ContractID, wl_.Name)
		return id == wl.ID
	})

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
	current, err := s.hasEnoughCapacity(wl)
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

func (s *statsStream) Workloads() (int, error) {
	capacity, err := s.stats.storage.Capacity()
	if err != nil {
		return 0, err
	}
	return capacity.Workloads, nil
}

func (s *statsStream) GetCounters() (pkg.Counters, error) {
	activeCounters, err := s.stats.active()
	if err != nil {
		return pkg.Counters{}, err
	}

	reserved, err := s.stats.reserved()
	if err != nil {
		return pkg.Counters{}, err
	}
	return pkg.Counters{
		Total:  s.stats.Total(),
		Used:   activeCounters.cap,
		System: reserved,
		Users: pkg.UsersCounters{
			Deployments:             activeCounters.deployments,
			Workloads:               activeCounters.workloads,
			LastDeploymentTimestamp: activeCounters.lastDeploymentTimestamp,
		},
	}, nil
}

func (s *statsStream) ListGPUs() ([]pkg.GPUInfo, error) {
	usedGpus := func() (map[string]uint64, error) {
		gpus := make(map[string]uint64)
		active, err := s.stats.storage.Capacity()
		if err != nil {
			return nil, err
		}
		for _, dl := range active.Deployments {
			for _, wl := range dl.Workloads {
				if wl.Type != zos.ZMachineType {
					continue
				}
				var vm zos.ZMachine
				if err := json.Unmarshal(wl.Data, &vm); err != nil {
					return nil, errors.Wrapf(err, "invalid workload data (%d.%s)", dl.ContractID, wl.Name)
				}

				for _, gpu := range vm.GPU {
					gpus[string(gpu)] = dl.ContractID
				}
			}
		}
		return gpus, nil
	}
	var list []pkg.GPUInfo
	if kernel.GetParams().IsGPUDisabled() {
		return list, nil
	}
	devices, err := capacity.ListPCI(capacity.GPU)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list available devices")
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to list active deployments")
	}

	used, err := usedGpus()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list used gpus")
	}

	for _, pciDevice := range devices {
		id := pciDevice.ShortID()
		info := pkg.GPUInfo{
			ID:       id,
			Vendor:   "unknown",
			Device:   "unknown",
			Contract: used[id],
		}

		vendor, device, ok := pciDevice.GetDevice()
		if ok {
			info.Vendor = vendor.Name
			info.Device = device.Name
		}

		subdevice, ok := pciDevice.GetSubdevice()
		if ok {
			info.Device = subdevice.Name
		}

		list = append(list, info)
	}

	return list, nil
}
