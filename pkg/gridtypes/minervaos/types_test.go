package zos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestWorkloadData(t *testing.T) {
	require := require.New(t)

	wl := gridtypes.Workload{
		Type: ZMountType,
		Data: json.RawMessage(`{"size": 10}`),
	}

	data, err := wl.WorkloadData()
	require.NoError(err)

	require.IsType(&ZMount{}, data)
	volume := data.(*ZMount)

	require.Equal(gridtypes.Unit(10), volume.Size)
}

func TestWorkloadValidation(t *testing.T) {
	require := require.New(t)

	wl := gridtypes.Workload{
		Type: ZMountType,
		Name: "name",
		Data: json.RawMessage(`{"size": 10}`),
	}

	err := wl.Valid(nil)
	require.NoError(err)

}
