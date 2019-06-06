package upgrade

import (
	"fmt"
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
	assert.True(t, IsExecutable(stat.Mode()))

	f, err = os.OpenFile(filepath.Join(dir, "normal"), os.O_CREATE|os.O_RDONLY|os.O_EXCL, 0440)
	require.NoError(t, err)

	stat, err = f.Stat()
	require.NoError(t, err)
	assert.False(t, IsExecutable(stat.Mode().Perm()))
}

func TestListDir(t *testing.T) {
	files, err := listDir("/etc")
	require.NoError(t, err)
	fmt.Println(files)
}
