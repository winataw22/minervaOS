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

func TestHexEncoder(t *testing.T) {
	txt := `{"value":"0011ccff"}`

	var loaded struct {
		Value Bytes `json:"value"`
	}

	err := json.Unmarshal([]byte(txt), &loaded)
	require.NoError(t, err)

	require.Equal(t, Bytes{0x00, 0x11, 0xcc, 0xff}, loaded.Value)
	ser, err := json.Marshal(loaded)
	require.NoError(t, err)
	require.Equal(t, []byte(txt), ser)
}
