package upgrade

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsExecutable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	f, err := os.OpenFile(filepath.Join(dir, "exec"), os.O_CREATE|os.O_RDONLY|os.O_EXCL, 0770)
	require.NoError(t, err)

	stat, err := f.Stat()
	require.NoError(t, err)
	assert.True(t, isExecutable(stat.Mode()))

	f, err = os.OpenFile(filepath.Join(dir, "normal"), os.O_CREATE|os.O_RDONLY|os.O_EXCL, 0440)
	require.NoError(t, err)

	stat, err = f.Stat()
	require.NoError(t, err)
	assert.False(t, isExecutable(stat.Mode().Perm()))
}

func TestTrimMounpoint(t *testing.T) {
	for _, tc := range []struct {
		path       string
		mountpoint string
		result     string
	}{
		{
			path:       "/mnt/foo/bin/bar",
			mountpoint: "/mnt/foo",
			result:     "/bin/bar",
		},
		{
			path:       "/mnt/foo/bin/bar",
			mountpoint: "/mnt/foo/",
			result:     "/bin/bar",
		},
	} {
		result := trimMounpoint(tc.mountpoint, tc.path)
		assert.Equal(t, tc.result, result)
	}
}
