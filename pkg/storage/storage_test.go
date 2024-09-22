package storage

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
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
	name  string
	usage filesystem.Usage
	ptype zos.DeviceType
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

func (p *testPool) Mounted() (string, error) {
	return p.Path(), nil
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

func (p *testPool) Type() (zos.DeviceType, bool, error) {
	return p.ptype, false, nil
}

func (p *testPool) SetType(_ zos.DeviceType) error {
	return nil
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

func (p *testPool) Device() filesystem.DeviceInfo {
	return filesystem.DeviceInfo{}
}

func (p *testPool) Shutdown() error {
	return nil
}

func TestCreateSubvol(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 300,
		},
		ptype: zos.SSDDevice,
	}

	pool2 := &testPool{
		name: "pool-2",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
			pool1, pool2,
		},
	}

	sub := &testVolume{
		name: "sub",
	}

	pool2.On("AddVolume", "sub").Return(sub, nil)
	sub.On("Limit", uint64(500)).Return(nil)

	pool1.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.createSubvolWithQuota(500, "sub", PolicySSDOnly)

	require.NoError(err)
}

func TestCreateSubvolUnlimited(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 200,
		},
		ptype: zos.SSDDevice,
	}

	pool2 := &testPool{
		name: "pool-2",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool3 := &testPool{
		name: "pool-3",
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: zos.HDDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
			pool1, pool2,
		},
		hdds: []filesystem.Pool{
			pool3,
		},
	}

	sub := &testVolume{
		name: "sub",
	}

	pool2.On("AddVolume", "sub").Return(sub, nil)
	sub.On("Limit", uint64(0)).Return(nil)

	pool1.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool2.On("Volumes").Return([]filesystem.Volume{}, nil)
	pool3.On("Volumes").Return([]filesystem.Volume{}, nil)

	_, err := mod.createSubvolWithQuota(0, "sub", PolicySSDOnly)

	require.NoError(err)
}

func TestCreateSubvolNoSpaceLeft(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool2 := &testPool{
		name: "pool-2",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool3 := &testPool{
		name: "pool-3",
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: zos.HDDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
			pool1, pool2,
		},
		hdds: []filesystem.Pool{
			pool3,
		},
	}

	// from the data above the create subvol will prefer pool 2 because it
	// after adding the subvol, it will still has more space.

	_, err := mod.createSubvolWithQuota(20000, "sub", PolicySSDOnly)

	require.EqualError(err, "not enough space left in pools of this type ssd")
}

func TestVDiskFindCandidatesHasEnoughSpace(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool2 := &testPool{
		name: "pool-2",
		usage: filesystem.Usage{
			Size: 10000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool3 := &testPool{
		name: "pool-3",
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: zos.HDDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
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

	_, err := mod.diskFindCandidate(500)

	require.NoError(err)
}

func TestVDiskFindCandidatesNoSpace(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 5000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool2 := &testPool{
		name: "pool-2",
		usage: filesystem.Usage{
			Size: 5000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	pool3 := &testPool{
		name: "pool-3",
		usage: filesystem.Usage{
			Size: 100000,
			Used: 0,
		},
		ptype: zos.SSDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
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

	_, err := mod.diskFindCandidate(10000)
	require.NoError(err)

	if ok := pool3.AssertCalled(t, "AddVolume", vdiskVolumeName); !ok {
		t.Fail()
	}
}

func TestVDiskFindCandidatesOverProvision(t *testing.T) {
	require := require.New(t)

	pool1 := &testPool{
		name: "pool-1",
		usage: filesystem.Usage{
			Size: 5000,
			Used: 100,
		},
		ptype: zos.SSDDevice,
	}

	mod := Module{
		ssds: []filesystem.Pool{
			pool1,
		},
	}

	sub := &testVolume{
		name: vdiskVolumeName,
	}

	pool1.On("Volumes").Return([]filesystem.Volume{sub}, nil)

	_, err := mod.diskFindCandidate(4000)
	require.NoError(err)

	_, err = mod.diskFindCandidate(5000)
	require.Error(err)

}

func TestCacheResize(t *testing.T) {
	// resize down
	var m Module
	cacheSize := uint64(5)

	vol := testVolume{
		usage: filesystem.Usage{
			Size: 100,
			Used: 100,
			Excl: 1,
		},
	}
	vol.On("Limit", cacheSize).Return(nil)
	err := m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)

	vol = testVolume{
		usage: filesystem.Usage{
			Size: 100,
			Used: 100,
			Excl: 19,
		},
	}
	// the limit is then set to double the 19
	// = 19 * 2 = 38
	// this then is ceiled to multiple of cacheSize
	// so  (38/5)* 5 = 35
	// then 35 + 5 = 40
	vol.On("Limit", uint64(40)).Return(nil)
	err = m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)

	// resize down
	vol = testVolume{
		usage: filesystem.Usage{
			Size: 100,
			Used: 100,
			Excl: 0, // no files
		},
	}
	vol.On("Limit", cacheSize).Return(nil)
	err = m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)

	// resize down
	vol = testVolume{
		usage: filesystem.Usage{
			Size: 1000,
			Used: 1000,
			Excl: 16,
		},
	}
	vol.On("Limit", uint64(35)).Return(nil)
	err = m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)

	// resize up
	vol = testVolume{
		usage: filesystem.Usage{
			Size: 100,
			Used: 100,
			Excl: 91,
		},
	}
	vol.On("Limit", 100+cacheSize).Return(nil)
	err = m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)

	// leave as is
	vol = testVolume{
		usage: filesystem.Usage{
			Size: 100,
			Used: 100,
			Excl: 50,
		},
	}
	err = m.checkAndResizeCache(&vol, cacheSize)
	require.NoError(t, err)
}
