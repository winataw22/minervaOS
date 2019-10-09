package filesystem

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeviceManagerScan(t *testing.T) {
	require := require.New(t)
	var exec TestExecuter
	ctx := context.Background()

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	devices.BlockDevices = []Device{
		{Type: "block", Path: "/tmp/dev1", Label: "test"},
		{Type: "block", Path: "/tmp/dev2", Children: []Device{
			{Path: "/tmp/dev2-1"},
			{Path: "/tmp/dev2-2"},
		}},
	}

	bytes, err := json.Marshal(devices)
	require.NoError(err)

	// we expect this call to lsblk
	exec.On("run", ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2,11", "--path").
		Return(bytes, nil)

	// then the devices will be tested for types (per device)
	exec.On("run", mock.Anything, "seektime", "-j", "/tmp/dev1").
		Return([]byte(`{"type": "SSD", "elapsed": 100}`), nil)

	exec.On("run", mock.Anything, "seektime", "-j", "/tmp/dev2").
		Return([]byte(`{"type": "HDD", "elapsed": 5000}`), nil)

	mgr, err := defaultDeviceManager(ctx, &exec)
	require.NoError(err)

	cached, err := mgr.Devices(ctx)
	require.NoError(err)

	require.Len(cached, 4)
	// make sure all types are set.
	for _, dev := range cached {
		require.NotEmpty(dev.DiskType, "device: %s", dev.Path)
	}

	filtered, err := mgr.ByLabel(ctx, "test")
	require.NoError(err)
	require.Len(filtered, 1)

	require.Equal("/tmp/dev1", cached[0].Path)

}
