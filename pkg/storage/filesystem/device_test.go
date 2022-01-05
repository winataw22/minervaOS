package filesystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeviceManagerScan(t *testing.T) {
	require := require.New(t)
	var exec TestExecuter
	ctx := context.Background()

	// we expect this call to lsblk
	exec.On("run", ctx, "lsblk", "--json", "-o", "PATH,NAME,SIZE,SUBSYSTEMS,FSTYPE,LABEL", "--bytes", "--exclude", "1,2,11", "--path").
		Return(TestMap{
			"blockdevices": []TestMap{
				{"subsystems": "block:scsi:pci", "path": "/tmp/dev1", "name": "dev1"},
				{"subsystems": "block:scsi:pci", "path": "/tmp/dev2", "name": "dev2"},
			},
		}.Bytes(), nil)

	// then other calls per device for extended details
	exec.On("run", ctx, "lsblk", "--json", "-o", "PATH,NAME,SIZE,SUBSYSTEMS,FSTYPE,LABEL", "--bytes", "--exclude", "1,2,11", "--path", "/tmp/dev1").
		Return(TestMap{
			"blockdevices": []TestMap{
				{"subsystems": "block:scsi:pci", "path": "/tmp/dev1", "name": "dev1", "label": "test"},
			},
		}.Bytes(), nil)

	exec.On("run", ctx, "lsblk", "--json", "-o", "PATH,NAME,SIZE,SUBSYSTEMS,FSTYPE,LABEL", "--bytes", "--exclude", "1,2,11", "--path", "/tmp/dev2").
		Return(TestMap{
			"blockdevices": []TestMap{
				{"subsystems": "block:scsi:pci", "path": "/tmp/dev2", "name": "dev2", "label": "test2"},
			},
		}.Bytes(), nil)

	exec.On("run", ctx, "findmnt", "-J", "-S", "/tmp/dev1").
		Return(TestMap{
			"filesystems": []TestMap{
				{"source": "/tmp/dev1", "target": "", "optoins": ""},
			},
		}.Bytes(), nil)

	exec.On("run", ctx, "findmnt", "-J", "-S", "/tmp/dev2").
		Return(TestMap{
			"filesystems": []TestMap{
				{"source": "/tmp/dev2", "target": "", "optoins": ""},
			},
		}.Bytes(), nil)

	exec.On("run", ctx, "findmnt", "-J").
		Return(TestMap{
			"filesystems": []TestMap{
				{"source": "/tmp/dev1", "target": "", "options": ""},
				{"source": "/tmp/dev2", "target": "", "options": ""},
			},
		}.Bytes(), nil)

	// then the devices will be tested for types (per device)
	exec.On("run", mock.Anything, "seektime", "-j", "/tmp/dev1").
		Return([]byte(`{"type": "SSD", "elapsed": 100}`), nil)

	exec.On("run", mock.Anything, "seektime", "-j", "/tmp/dev2").
		Return([]byte(`{"type": "HDD", "elapsed": 5000}`), nil)

	mgr := defaultDeviceManager(ctx, &exec)

	cached, err := mgr.Devices(ctx)
	require.NoError(err)

	require.Len(cached, 2)
	// make sure all types are set.
	for _, dev := range cached {
		require.NotEmpty(dev.Type(), "device: %s", dev.Path)
	}

	filtered, err := mgr.ByLabel(ctx, "test")
	require.NoError(err)
	require.Len(filtered, 1)

	require.Equal("/tmp/dev1", filtered[0].Path())

}
