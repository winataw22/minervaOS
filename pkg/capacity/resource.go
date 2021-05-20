package capacity

import (
	"context"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func (r *ResourceOracle) cru() (uint64, error) {
	n, err := cpu.Counts(true)
	return uint64(n), err
}

func (r *ResourceOracle) mru() (gridtypes.Unit, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	// we round the value to nearest Gigabyte
	return gridtypes.Unit(vm.Total), nil
}

func (r *ResourceOracle) sru() (gridtypes.Unit, error) {
	total, err := r.storage.Total(context.TODO(), zos.SSDDevice)
	if err != nil {
		return 0, err
	}

	return gridtypes.Unit(total), nil
}

func (r *ResourceOracle) hru() (gridtypes.Unit, error) {
	total, err := r.storage.Total(context.TODO(), zos.HDDDevice)
	if err != nil {
		return 0, err
	}

	return gridtypes.Unit(total), nil
}
