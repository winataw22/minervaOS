package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type Volume struct {
	Size gridtypes.Unit `json:"size"`
}

var _ gridtypes.WorkloadData = (*Volume)(nil)

func (v Volume) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{
		SRU: v.Size,
	}, nil
}

func (v Volume) Valid(getter gridtypes.WorkloadGetter) error {
	if v.Size == 0 {
		return fmt.Errorf("invalid size")
	}

	return nil
}

func (v Volume) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", v.Size); err != nil {
		return err
	}

	return nil
}

type VolumeResult struct {
	ID string `json:"id"`
}
