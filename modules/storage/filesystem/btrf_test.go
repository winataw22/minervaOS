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
	"strings"
	"testing"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
)

type TestDevices map[string]string

func (d TestDevices) Loops() []Device {
	var loops []Device
	for _, loop := range d {
		loops = append(loops, Device{Path: loop})
	}

	return loops
}

func (d TestDevices) Destroy() {
	for file, loop := range d {
		if loop != "" {
			run(context.Background(), "losetup", "-d", loop)
		}

		if err := os.Remove(file); err != nil {
			fmt.Printf("failed to delete file '%s': %s\n", file, err)
		}
	}
}

func setupDevices(count int) (devices TestDevices, err error) {
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

type TestDeviceManager struct {
	devices TestDevices
}

func (m TestDeviceManager) Device(ctx context.Context, device string) (*Device, error) {
	for _, loop := range m.devices {
		if loop == device {
			return &Device{
				Path: loop,
				Type: "loop",
			}, nil
		}
	}

	return nil, fmt.Errorf("device not found")
}

func (m TestDeviceManager) ByLabel(ctx context.Context, label string) (DeviceCache, error) {
	return nil, nil
}

func (m TestDeviceManager) Devices(ctx context.Context) (DeviceCache, error) {
	var devices DeviceCache
	for _, loop := range m.devices {
		devices = append(devices, &Device{
			Path: loop,
			Type: "loop",
		})
	}

	return devices, nil
}

func (m TestDeviceManager) Scan(ctx context.Context) error {
	return nil
}

func TestMain(m *testing.M) {
	defer func() {
		//make sure to try to detach all remaining loop devices from testing
		run(context.Background(), "losetup", "-D")
	}()

	os.Exit(m.Run())
}

func basePoolTest(t *testing.T, pool Pool) {
	t.Run("test mounted", func(t *testing.T) {
		_, mounted := pool.Mounted()
		if ok := assert.False(t, mounted); !ok {
			t.Error()
		}
	})

	t.Run("test path", func(t *testing.T) {
		if ok := assert.Equal(t, path.Join("/mnt", pool.Name()), pool.Path()); !ok {
			t.Error()
		}
	})

	t.Run("test mount", func(t *testing.T) {
		// mount device
		target, err := pool.Mount()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Equal(t, target, pool.Path()); !ok {
			t.Error()
		}
	})

	defer pool.UnMount()

	t.Run("test no subvolumes", func(t *testing.T) {
		// no volumes
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})

	var volume Volume
	var err error
	t.Run("test create volume", func(t *testing.T) {
		volume, err = pool.AddVolume("subvol1")
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Equal(t, path.Join("/mnt", pool.Name(), "subvol1"), volume.Path()); !ok {
			t.Error()
		}
	})

	t.Run("test list volumes", func(t *testing.T) {
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Len(t, volumes, 1); !ok {
			t.Error()
		}
	})

	t.Run("test subvolume list no subvolumes", func(t *testing.T) {
		volumes, err := volume.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})

	t.Run("test limit subvolume", func(t *testing.T) {
		usage, err := volume.Usage()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		// Note: an empty subvolume has an overhead of 16384 bytes
		if ok := assert.Equal(t, Usage{Used: 16384}, usage); !ok {
			t.Fail()
		}

		err = volume.Limit(50 * 1024 * 1024)
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		usage, err = volume.Usage()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		// Note: an empty subvolume has an overhead of 16384 bytes
		if ok := assert.Equal(t, Usage{Used: 16384, Size: 50 * 1024 * 1024}, usage); !ok {
			t.Fail()
		}
	})

	t.Run("test remove subvolume", func(t *testing.T) {
		err = pool.RemoveVolume("subvol1")
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}
		// no volumes after delete
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})
}

func TestBtrfsSingle(t *testing.T) {
	devices, err := setupDevices(1)
	if err != nil {
		t.Fatal("failed to initialize devices", err)
	}
	defer devices.Destroy()

	fs := NewBtrfs(TestDeviceManager{devices})
	devs := DeviceCache{}
	for idx := range devices.Loops() {
		devs = append(devs, &devices.Loops()[idx])
	}
	pool, err := fs.Create(context.Background(), "test-single", devs, modules.Single)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	basePoolTest(t, pool)
}

func TestBtrfsRaid1(t *testing.T) {
	devices, err := setupDevices(3)
	if err != nil {
		t.Fatal("failed to initialize devices", err)
	}

	defer devices.Destroy()

	loops := devices.Loops()
	fs := NewBtrfs(TestDeviceManager{devices})
	devs := DeviceCache{}
	for idx := range devices.Loops()[:2] {
		devs = append(devs, &devices.Loops()[idx])
	}
	pool, err := fs.Create(context.Background(), "test-raid1", devs, modules.Raid1) //use the first 2 disks

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	basePoolTest(t, pool)

	//make sure pool is mounted
	_, err = pool.Mount()
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	defer pool.UnMount()

	// raid  specific tests

	t.Run("add device", func(t *testing.T) {
		// add a device to array
		err = pool.AddDevice(&loops[2])
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}
	})

	t.Run("remove device", func(t *testing.T) {
		// remove device from array
		err = pool.RemoveDevice(&loops[0])
		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}
	})

	t.Run("remove second device", func(t *testing.T) {
		// remove a 2nd device should fail because raid1 should
		// have at least 2 devices
		err = pool.RemoveDevice(&loops[1])
		if ok := assert.Error(t, err); !ok {
			t.Fatal()
		}
	})
}

func TestBtrfsList(t *testing.T) {
	devices, err := setupDevices(2)
	if err != nil {
		t.Fatal("failed to initialize devices", err)
	}

	defer devices.Destroy()

	fs := NewBtrfs(TestDeviceManager{devices})

	loops := devices.Loops()
	names := make(map[string]struct{})
	for i, loop := range loops {
		name := fmt.Sprintf("test-list-%d", i)
		names[name] = struct{}{}
		_, err := fs.Create(context.Background(), name, DeviceCache{&loop}, modules.Single)

		if ok := assert.NoError(t, err); !ok {
			t.Fatal()
		}
	}

	pools, err := fs.List(context.Background())
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	for _, pool := range pools {
		if !strings.HasPrefix(pool.Name(), "test-list") {
			continue
		}

		_, exist := names[pool.Name()]
		if !exist {
			t.Fatalf("pool %s is not listed", pool)
		}

		delete(names, pool.Name())
	}

	if ok := assert.Len(t, names, 0); !ok {
		t.Fatal("not all pools were listed")
	}
}
