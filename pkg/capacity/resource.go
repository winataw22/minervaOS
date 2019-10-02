package capacity

import (
	"math"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg"
)

func (r *ResourceOracle) cru() (uint64, error) {
	n, err := cpu.Counts(true)
	return uint64(n), err
}

func (r *ResourceOracle) mru() (uint64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	total := float64(vm.Total) / float64(GiB)
	return uint64(math.Round(total)), nil
}

func (r *ResourceOracle) sru() (uint64, error) {
	total, err := r.storage.Total(pkg.SSDDevice)
	if err != nil {
		return 0, err
	}

	return total / GiB, nil
}

func (r *ResourceOracle) hru() (uint64, error) {
	total, err := r.storage.Total(pkg.HDDDevice)
	if err != nil {
		return 0, err
	}

	return total / GiB, nil
}
