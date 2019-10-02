package backend

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockOperations(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// create a dummy file to lock
	path := filepath.Join(dir, "x")
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	// now use it to lock
	m, err := NewFileLock(path)
	require.NoError(t, err)

	err = m.Lock()
	require.NoError(t, err)
	err = m.Unlock()
	require.NoError(t, err)
}

func TestLockFolderPath(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// use the folder to lock
	m, err := NewFileLock(dir)
	require.NoError(t, err)

	err = m.Lock()
	require.NoError(t, err)
	err = m.Unlock()
	require.NoError(t, err)
}
