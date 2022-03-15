package filesystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type TestDeviceManager struct {
	mock.Mock
}

// Device returns the device at the specified path
func (m *TestDeviceManager) Device(ctx context.Context, device string) (DeviceInfo, error) {
	args := m.Called(ctx, device)
	return args.Get(0).(DeviceInfo), args.Error(1)
}

// Devices finds all devices on a system
func (m *TestDeviceManager) Devices(ctx context.Context) (Devices, error) {
	args := m.Called(ctx)
	return args.Get(0).(Devices), args.Error(1)
}

// ByLabel finds all devices with the specified label
func (m *TestDeviceManager) ByLabel(ctx context.Context, label string) (Devices, error) {
	args := m.Called(ctx, label)
	return args.Get(0).(Devices), args.Error(1)
}

// Mountpoint returns mount point of a device
func (m *TestDeviceManager) Mountpoint(ctx context.Context, device string) (string, error) {
	args := m.Called(ctx, device)
	return args.String(0), args.Error(1)
}

func (m *TestDeviceManager) Seektime(ctx context.Context, device string) (zos.DeviceType, error) {
	args := m.Called(ctx, device)
	return zos.DeviceType(args.String(0)), args.Error(1)
}

func TestBtrfsCreatePoolExists(t *testing.T) {
	require := require.New(t)
	exe := &TestExecuter{}

	mgr := TestDeviceManager{}
	mgr.On("Mountpoint", mock.Anything, "/tmp/disk").
		Return("/mnt/some-label", nil)
	dev := DeviceInfo{
		mgr:        &mgr,
		Path:       "/tmp/disk",
		Rota:       false,
		Label:      "some-label",
		Filesystem: BtrfsFSType,
	}
	pool, err := newBtrfsPool(dev, exe)
	require.NoError(err)
	require.NotNil(pool)

	mnt, err := pool.Mounted()
	require.NoError(err)
	require.Equal("/mnt/some-label", mnt)

	mgr = TestDeviceManager{}
	mgr.On("Mountpoint", mock.Anything, "/tmp/disk").
		Return("", nil)
	dev = DeviceInfo{
		mgr:        &mgr,
		Path:       "/tmp/disk",
		Rota:       false,
		Label:      "some-label",
		Filesystem: BtrfsFSType,
	}
	// if not mounted!
	pool, err = newBtrfsPool(dev, exe)
	require.NoError(err)
	require.NotNil(pool)

	_, err = pool.Mounted()
	require.ErrorIs(err, ErrDeviceNotMounted)
}

func TestBtrfsCreatePoolNotExist(t *testing.T) {
	// this should actually create a btrfs pool
	require := require.New(t)

	exe := &TestExecuter{}
	dev := DeviceInfo{
		Path: "/tmp/disk",
		Rota: false,
	}

	// expected formating of the device
	ctx := context.Background()
	exe.On("run", ctx, "mkfs.btrfs", "-L", mock.AnythingOfType("string"), dev.Path).
		Return([]byte{}, nil)

	pool, err := newBtrfsPool(dev, exe)
	require.NoError(err)
	require.NotNil(pool)
}
