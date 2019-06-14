package flist

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
)

func testFlistModule(t *testing.T) (modules.Flister, func()) {
	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(root)
	}
	return New(root), cleanup
}
func TestMountUmount(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	f, cleanup := testFlistModule(t)
	defer cleanup()

	path, err := f.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	// check file are accessible
	assert.NotEqual("", path)
	infos, err := ioutil.ReadDir(path)
	require.NoError(err)
	assert.True(len(infos) > 0)

	// check pid file exsists
	dir, name := filepath.Split(path)
	dir, _ = filepath.Split(dir[:len(dir)-1])
	pidPath := filepath.Join(dir, "pid", name) + ".pid"
	_, err = os.Stat(pidPath)
	assert.NoError(err)

	// try to unmount other part of the filesystem
	err = f.Umount("/mnt/media/disk1")
	assert.Error(err)

	err = f.Umount(path)
	assert.NoError(err)

	// check pid file is gone after unmount
	_, err = os.Stat(pidPath)
	assert.Error(err)
	assert.True(os.IsNotExist(err))
}

func TestIsolation(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	f, cleanup := testFlistModule(t)
	defer cleanup()

	path1, err := f.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	path2, err := f.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	defer f.Umount(path1)
	defer f.Umount(path2)

	t.Run("new file isolated", func(t *testing.T) {
		err = ioutil.WriteFile(filepath.Join(path1, "newfile"), []byte("hello world"), 0660)
		require.NoError(err)

		_, err = os.Open(filepath.Join(path2, "newfile"))
		assert.True(os.IsNotExist(err))
	})

	t.Run("common file isolation", func(t *testing.T) {
		targetFile := "etc/resolv.conf"
		content := []byte("nameserver 1.1.1.1")
		err = ioutil.WriteFile(filepath.Join(path1, targetFile), content, 0660)
		require.NoError(err)

		b, err := ioutil.ReadFile(filepath.Join(path2, targetFile))
		require.NoError(err)
		assert.NotEqual(content, b)
	})
}

func TestDownloadFlist(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	x, cleanup := testFlistModule(t)
	defer cleanup()

	f, ok := x.(*flistModule)
	require.True(ok)
	path1, err := f.downloadFlist("https://hub.grid.tf/zaibon/ubuntu_bionic.flist")
	require.NoError(err)

	info1, err := os.Stat(path1)
	require.NoError(err)

	path2, err := f.downloadFlist("https://hub.grid.tf/zaibon/ubuntu_bionic.flist")
	require.NoError(err)

	assert.Equal(path1, path2)

	// mod time should be the same, this proof the second download
	// didn't actually re-wrote the file a second time
	info2, err := os.Stat(path2)
	require.NoError(err)
	assert.Equal(info1.ModTime(), info2.ModTime())
}
