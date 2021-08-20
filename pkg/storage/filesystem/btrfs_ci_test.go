/*
NOTE:
  This test file tries to create loop devices to work against on some
  sparse files, to avoid causing any actual changes to permanent storage
  on the test machine. This comes with a price that this test file
  need to run as root to be able to run the `losetup` commands.

  this can be easily done as
  `sudo GOPATH=$GOPATH go test -v ./...`

   (
	  we set the gopath to the current user gopath to make it use the same go cache and src from the
	  current user
   )
*/
package filesystem

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var (
	SkipCITests = false
)

type TestDevices map[string]string

func (d TestDevices) Loops() Devices {
	mgr := lsblkDeviceManager{
		executer: executerFunc(run),
	}
	for _, loop := range d {
		mgr.cache = append(mgr.cache, minDevice{
			IPath:    loop,
			IName:    filepath.Base(loop),
			DiskType: zos.SSDDevice,
		})
	}

	devices, err := mgr.Devices(context.Background())
	if err != nil {
		panic(err)
	}
	return devices
}

func (d TestDevices) Destroy() {
	for file, loop := range d {
		if loop != "" {
			_, _ = run(context.Background(), "losetup", "-d", loop)
		}

		if err := os.Remove(file); err != nil {
			fmt.Printf("failed to delete file '%s': %s\n", file, err)
		}
	}
}

func SetupDevices(count int) (devices TestDevices, err error) {
	devices = make(TestDevices)

	defer func() {
		if err != nil {
			devices.Destroy()
		}
	}()

	for i := 0; i < count; i++ {
		var dev *os.File
		dev, err = ioutil.TempFile("", "loop-test-")
		if err != nil {
			return
		}

		devices[dev.Name()] = ""

		if err = dev.Truncate(1024 * 1024 * 1024); err != nil { // 1G
			return
		}

		if err = dev.Close(); err != nil {
			return
		}
		var output []byte
		output, err = run(context.Background(), "losetup", "-f", "--show", dev.Name())
		if err != nil {
			return
		}

		devices[dev.Name()] = strings.TrimSpace(string(output))
	}

	return
}

func TestMain(m *testing.M) {
	devices, err := SetupDevices(1)
	if err != nil {
		// we can't create devices. we need to skip
		// CI test
		fmt.Println("skipping ci tests")
		SkipCITests = true
	} else {
		devices.Destroy()
	}

	defer func() {
		//make sure to try to detach all remaining loop devices from testing
		_, _ = run(context.Background(), "losetup", "-D")
	}()

	os.Exit(m.Run())
}

func basePoolTest(t *testing.T, pool Pool) {
	t.Run("test mounted", func(t *testing.T) {
		_, err := pool.Mounted()
		assert.ErrorIs(t, err, ErrDeviceNotMounted)
	})

	t.Run("test path", func(t *testing.T) {
		assert.Equal(t, path.Join("/mnt", pool.Name()), pool.Path())
	})

	t.Run("test mount", func(t *testing.T) {
		// mount device
		target, err := pool.Mount()
		assert.NoError(t, err)

		assert.Equal(t, target, pool.Path())
	})

	defer func() {
		_ = pool.UnMount()
	}()

	t.Run("test no subvolumes", func(t *testing.T) {
		// no volumes
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Empty(t, volumes)
	})

	var volume Volume
	var err error
	t.Run("test create volume", func(t *testing.T) {
		volume, err = pool.AddVolume("subvol1")
		require.NoError(t, err)

		assert.Equal(t, path.Join("/mnt", pool.Name(), "subvol1"), volume.Path())
	})

	t.Run("test list volumes", func(t *testing.T) {
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Len(t, volumes, 1)
	})

	t.Run("test usage", func(t *testing.T) {
		usage, err := pool.Usage()
		require.NoError(t, err)
		assert.Equal(t, uint64(1024*1024*1024), usage.Size)
	})

	t.Run("test limit subvolume", func(t *testing.T) {
		usage, err := volume.Usage()
		require.NoError(t, err)

		// Note: an empty subvolume has an overhead of 16384 bytes
		assert.Equal(t, Usage{Used: 16384}, usage)

		err = volume.Limit(50 * 1024 * 1024)
		require.NoError(t, err)

		usage, err = volume.Usage()
		require.NoError(t, err)

		// Note: an empty subvolume has an overhead of 16384 bytes
		assert.Equal(t, Usage{Used: 16384, Size: 50 * 1024 * 1024}, usage)
	})

	t.Run("test remove subvolume", func(t *testing.T) {
		err = pool.RemoveVolume("subvol1")
		require.NoError(t, err)
		// no volumes after delete
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Empty(t, volumes)
	})
}

func TestBtrfsSingleCI(t *testing.T) {
	if SkipCITests {
		t.Skip("test requires ability to create loop devices")
	}

	devices, err := SetupDevices(1)
	require.NoError(t, err, "failed to initialize devices")

	defer devices.Destroy()
	loops := devices.Loops()

	for _, dev := range loops {
		pool, err := NewBtrfsPool(dev)
		require.NoError(t, err)
		basePoolTest(t, pool)
	}
}

func TestCLeanUpQgroupsCI(t *testing.T) {
	if SkipCITests {
		t.Skip("test requires ability to create loop devices")
	}

	devices, err := SetupDevices(1)
	require.NoError(t, err, "failed to initialize devices")
	defer devices.Destroy()

	loops := devices.Loops()
	pool, err := NewBtrfsPool(loops[0])
	require.NoError(t, err)

	_, err = pool.Mount()
	require.NoError(t, err)
	defer func() {
		_ = pool.UnMount()
	}()

	volume, err := pool.AddVolume("vol1")
	require.NoError(t, err)
	t.Logf("volume ID: %v\n", volume.ID())

	err = volume.Limit(256 * 1024 * 1024)
	require.NoError(t, err)

	btrfsVol, ok := volume.(*btrfsVolume)
	require.True(t, ok, "volume should be a btrfsVolume")

	qgroups, err := btrfsVol.utils.QGroupList(context.TODO(), pool.Path())
	require.NoError(t, err)
	assert.Equal(t, 1, len(qgroups))
	t.Logf("qgroups before delete: %v", qgroups)

	_, ok = qgroups[fmt.Sprintf("0/%d", btrfsVol.id)]
	assert.True(t, ok, "qgroups should contains a qgroup linked to the subvolume")

	err = pool.RemoveVolume("vol1")
	require.NoError(t, err)

	qgroups, err = btrfsVol.utils.QGroupList(context.TODO(), pool.Path())
	require.NoError(t, err)

	t.Logf("remaining qgroups: %+v", qgroups)
	assert.Equal(t, 0, len(qgroups), "qgroups should have been deleted with the subvolume")
}
