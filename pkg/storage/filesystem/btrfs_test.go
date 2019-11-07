package filesystem

import (
	"context"
	"fmt"
	"testing"

	"github.com/threefoldtech/zos/pkg"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestDeviceManager struct {
	devices DeviceCache
}

func (m *TestDeviceManager) Reset() DeviceManager {
	return m
}

func (m *TestDeviceManager) Device(ctx context.Context, path string) (*Device, error) {
	for idx := range m.devices {
		loop := &m.devices[idx]
		if loop.Path == path {
			return loop, nil
		}
	}

	return nil, fmt.Errorf("device not found")
}

func (m *TestDeviceManager) ByLabel(ctx context.Context, label string) ([]*Device, error) {
	var filterred []*Device
	for idx := range m.devices {
		device := &m.devices[idx]
		if device.Label == label {
			filterred = append(filterred, device)
		}
	}
	return filterred, nil
}

func (m *TestDeviceManager) Devices(ctx context.Context) (DeviceCache, error) {
	return m.devices, nil
}

func (m *TestDeviceManager) Raw(ctx context.Context) (DeviceCache, error) {
	return m.devices, nil
}

func TestBtrfsCreateSingle(t *testing.T) {
	require := require.New(t)
	mgr := &TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/dev1", DiskType: pkg.SSDDevice},
		},
	}

	var exec TestExecuter

	exec.On("run", mock.Anything, "mkfs.btrfs", "-L", "test-single", "-d", "single", "-m", "single", "/tmp/dev1").
		Return([]byte{}, nil)

	fs := newBtrfs(mgr, &exec)
	_, err := fs.Create(context.Background(), "test-single", pkg.Single, &mgr.devices[0])
	require.NoError(err)

	require.Equal("test-single", mgr.devices[0].Label)
	require.Equal(BtrfsFSType, mgr.devices[0].Filesystem)

	//basePoolTest(t, &exec, pool)
}

func TestBtrfsCreateRaid1(t *testing.T) {
	require := require.New(t)
	mgr := &TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/dev1", DiskType: pkg.SSDDevice},
			Device{Path: "/tmp/dev2", DiskType: pkg.SSDDevice},
		},
	}

	var exec TestExecuter

	exec.On("run", mock.Anything, "mkfs.btrfs", "-L", "test-raid1",
		"-d", "raid1", "-m", "raid1",
		"/tmp/dev1", "/tmp/dev2").Return([]byte{}, nil)

	fs := newBtrfs(mgr, &exec)
	_, err := fs.Create(context.Background(), "test-raid1", pkg.Raid1, &mgr.devices[0], &mgr.devices[1])
	require.NoError(err)

	require.Equal("test-raid1", mgr.devices[0].Label)
	require.Equal(BtrfsFSType, mgr.devices[0].Filesystem)

	require.Equal("test-raid1", mgr.devices[1].Label)
	require.Equal(BtrfsFSType, mgr.devices[1].Filesystem)
}
