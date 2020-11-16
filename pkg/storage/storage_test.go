package storage

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

type testVolume struct {
	mock.Mock
	name  string
	usage filesystem.Usage
}

func (p *testVolume) ID() int {
	return 0
}

func (p *testVolume) Path() string {
	return filepath.Join("/tmp", p.name)
}

func (p *testVolume) Usage() (filesystem.Usage, error) {
	return p.usage, nil
}

func (p *testVolume) Limit(size uint64) error {
	args := p.Called(size)
	return args.Error(0)
}

func (p *testVolume) Name() string {
	return p.name
}

func (p *testVolume) FsType() string {
	return "test"
}

type testPool struct {
	mock.Mock
	name     string
	usage    filesystem.Usage
	reserved uint64
	ptype    pkg.DeviceType
}

var _ filesystem.Pool = &testPool{}

func (p *testPool) ID() int {
	return 0
}

func (p *testPool) Path() string {
	return filepath.Join("/tmp", p.name)
}

func (p *testPool) Usage() (filesystem.Usage, error) {
	return p.usage, nil
}

func (p *testPool) Limit(_ uint64) error {
	return fmt.Errorf("not impelemneted")
}

func (p *testPool) Name() string {
	return p.name
}

func (p *testPool) FsType() string {
	return "test"
}

func (p *testPool) Mounted() (string, bool) {
	return p.Path(), true
}

func (p *testPool) Mount() (string, error) {
	return "", fmt.Errorf("Mount not implemented")
}

func (p *testPool) MountWithoutScan() (string, error) {
	return "", fmt.Errorf("MountWithoutScan not implemented")
}

func (p *testPool) UnMount() error {
	return fmt.Errorf("UnMount not implemented")
}

func (p *testPool) AddDevice(_ *filesystem.Device) error {
	return fmt.Errorf("AddDevice not implemented")
}

func (p *testPool) RemoveDevice(_ *filesystem.Device) error {
	return fmt.Errorf("RemoveDevice not implemented")
}

func (p *testPool) Type() pkg.DeviceType {
	return p.ptype
}

func (p *testPool) Reserved() (uint64, error) {
	return p.reserved, nil
}

func (p *testPool) Volumes() ([]filesystem.Volume, error) {
	args := p.Called()
	return args.Get(0).([]filesystem.Volume), args.Error(1)
}

func (p *testPool) AddVolume(name string) (filesystem.Volume, error) {
	args := p.Called(name)
	return args.Get(0).(filesystem.Volume), args.Error(1)
}

func (p *testPool) RemoveVolume(name string) error {
	args := p.Called(name)
	return args.Error(1)
}

func (p *testPool) Devices() []*filesystem.Device {
	return []*filesystem.Device{}
}

func (p *testPool) Shutdown() error {
	return nil
}

func TestCreateSubvol(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.HDDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	sub := &testVolume{
		name: "sub",
	}

	pool1.On("AddVolume", "sub").Return(sub, nil)
	sub.On("Limit", uint64(500)).Return(nil)

	pool1.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.createSubvolWithQuota(500, "sub", pkg.SSDDevice)

	require.NoError(err)
}

func TestCreateSubvolUnlimited(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.HDDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	sub := &testVolume{
		name: "sub",
	}

	pool1.On("AddVolume", "sub").Return(sub, nil)
	sub.On("Limit", uint64(0)).Return(nil)

	pool1.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.createSubvolWithQuota(0, "sub", pkg.SSDDevice)

	require.NoError(err)
}

func TestCreateSubvolNoSpaceLeft(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.HDDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	// from the data above the create subvol will prefer pool 2 because it
	// after adding the subvol, it will still has more space.

	_, err := mod.createSubvolWithQuota(20000, "sub", pkg.SSDDevice)

	require.EqualError(err, "Not enough space left in pools of this type ssd")
}

func TestVDiskFindCandidatesHasEnoughSpace(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.HDDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	sub := &testVolume{
		name: vdiskVolumeName,
	}

	// pool3.On("AddVolume", "sub").Return(sub, nil)
	// sub.On("Limit", uint64(0)).Return(nil)

	pool1.On("Volumes").Return([]filesystem.Volume{sub}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.VDiskFindCandidate(500)

	require.NoError(err)
}

func TestVDiskFindCandidatesWrongType(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.HDDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	sub := &testVolume{
		name: vdiskVolumeName,
	}

	pool3.On("AddVolume", vdiskVolumeName).Return(sub, nil)

	pool1.On("Volumes").Return([]filesystem.Volume{sub}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.VDiskFindCandidate(10000)
	require.EqualError(err, "Not enough space left in pools of this type ssd")

}

func TestVDiskFindCandidatesNoSpace(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name:     "pool-1",
		reserved: 2000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool2 := &testPool{
		name:     "pool-2",
		reserved: 1000,
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: pkg.SSDDevice,
	}

	pool3 := &testPool{
		name:     "pool-3",
		reserved: 0,
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: pkg.SSDDevice,
	}

	mod := Module{
		pools: []filesystem.Pool{
			pool1, pool2, pool3,
		},
	}

	sub := &testVolume{
		name: vdiskVolumeName,
	}

	pool3.On("AddVolume", vdiskVolumeName).Return(sub, nil)

	pool1.On("Volumes").Return([]filesystem.Volume{sub}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.VDiskFindCandidate(10000)
	require.NoError(err)

	if ok := pool3.AssertCalled(t, "AddVolume", vdiskVolumeName); !ok {
		t.Fail()
	}
}
