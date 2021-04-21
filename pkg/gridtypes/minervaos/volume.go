package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type DeviceType `json:"type"`
}

//Valid implements WorkloadData
func (v Volume) Valid(getter gridtypes.WorkloadGetter) error {
	if v.Size == 0 {
		return fmt.Errorf("invalid size")
	}

	if err := v.Type.Valid(); err != nil {
		return err
	}

	return nil
}

// Capacity implements WorkloadData
func (v Volume) Capacity() (cap gridtypes.Capacity, err error) {
	switch v.Type {
	case HDDDevice:
		cap.HRU = v.Size
	case SSDDevice:
		cap.SRU = v.Size
	default:
		return cap, fmt.Errorf("invalid volume type '%s'", v.Type.String())
	}

	return
}

// Challenge implements WorkloadData
func (v Volume) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", v.Size); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", string(v.Type)); err != nil {
		return err
	}

	return nil
}

// VolumeResult is the information return to the BCDB
// after deploying a volume
type VolumeResult struct {
	ID string `json:"volume_id"`
}
