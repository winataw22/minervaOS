package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// ZMount defines a mount point
type ZMount struct {
	// Size of the volume
	Size gridtypes.Unit `json:"size"`
}

// Valid implements WorkloadData
func (v ZMount) Valid(getter gridtypes.WorkloadGetter) error {
	if v.Size == 0 {
		return fmt.Errorf("invalid size")
	}

	return nil
}

// Capacity implements WorkloadData
func (v ZMount) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{
		SRU: v.Size,
	}, nil
}

// Challenge implements WorkloadData
func (v ZMount) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", v.Size); err != nil {
		return err
	}

	return nil
}

// ZMountResult is the information return to the BCDB
// after deploying a volume
type ZMountResult struct {
	ID string `json:"volume_id"`
}
