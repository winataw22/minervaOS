package filesystem

import (
	"context"
	"testing"

	"github.com/threefoldtech/zos/pkg/gridtypes/zos"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestDevice struct {
	path       string
	name       string
	size       uint64
	deviceType zos.DeviceType
	info       DeviceInfo
	readTime   uint64
}

// Path returns path to the device like /dev/sda
func (t *TestDevice) Path() string {
	return t.path
}

// Size device size
func (t *TestDevice) Size() uint64 {
	return t.size
}

// Name returns name of the device like sda
func (t *TestDevice) Name() string {
	return t.name
}

// Type returns detected device type (hdd, ssd)
func (t *TestDevice) Type() zos.DeviceType {
	return t.deviceType
}

// Info is current device information, this should not be cached because
// it might change over time
func (t *TestDevice) Info() (DeviceInfo, error) {
	return t.info, nil
}

// ReadTime detected read time of the device
func (t *TestDevice) ReadTime() uint64 {
	return t.readTime
}

func TestBtrfsCreatePoolExists(t *testing.T) {
	require := require.New(t)

	exe := &TestExecuter{}
	dev := TestDevice{
		path:       "/tmp/disk",
		name:       "disk",
		deviceType: zos.SSDDevice,
		readTime:   0,
		info: DeviceInfo{
			Path:       "/tmp/disk",
			Label:      "some-label",
			Mountpoint: "/mnt/some-label",
			Filesystem: BtrfsFSType,
		},
	}
	pool, err := newBtrfsPool(&dev, exe)
	require.NoError(err)
	require.NotNil(pool)

	mnt, err := pool.Mounted()
	require.NoError(err)
	require.Equal("/mnt/some-label", mnt)

	// if not mounted!
	dev.info.Mountpoint = ""
	pool, err = newBtrfsPool(&dev, exe)
	require.NoError(err)
	require.NotNil(pool)

	_, err = pool.Mounted()
	require.ErrorIs(err, ErrDeviceNotMounted)
}

func TestBtrfsCreatePoolNotExist(t *testing.T) {
	// this should actually create a btrfs pool
	require := require.New(t)

	exe := &TestExecuter{}
	dev := TestDevice{
		path:       "/tmp/disk",
		name:       "disk",
		deviceType: zos.SSDDevice,
		readTime:   0,
		info: DeviceInfo{
			Path: "/tmp/disk",
		},
	}

	// expected formating of the device
	ctx := context.Background()
	exe.On("run", ctx, "mkfs.btrfs", "-L", mock.AnythingOfType("string"), dev.path).
		Return([]byte{}, nil)

	pool, err := newBtrfsPool(&dev, exe)
	require.NoError(err)
	require.NotNil(pool)
}
