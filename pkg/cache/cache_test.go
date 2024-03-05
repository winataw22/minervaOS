package cache

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

// TestVolatileDir tests the volatileDir function against multiple scenarios.
// it tests both scenarios of the volatileDir failed/succeeded to be mounted
func TestVolatileDir(t *testing.T) {
	name := "test"
	var size uint64 = 1024
	const volatileBaseDir = "/var/run/cache"

	t.Run("volatileDir failed to Mount", func(t *testing.T) {
		os := &pkg.SystemOSMock{}

		filePath := filepath.Join(volatileBaseDir, name)

		os.On("MkdirAll", volatileBaseDir, fs.FileMode(0700)).
			Return(nil)

		os.On("Mkdir", filePath, fs.FileMode(0700)).
			Return(nil)

		os.On("Mount", "none", filePath, "tmpfs", uintptr(0), fmt.Sprintf("size=%d", size)).
			Return(fmt.Errorf("failed to Mount"))

		_, err := volatileDir(name, size, os, os)
		require.Error(t, err)
		os.AssertExpectations(t)
	})
	t.Run("volatileDir Mounted successfully", func(t *testing.T) {
		os := &pkg.SystemOSMock{}

		filePath := filepath.Join(volatileBaseDir, name)

		os.On("MkdirAll", volatileBaseDir, fs.FileMode(0700)).
			Return(nil)

		os.On("Mkdir", filePath, fs.FileMode(0700)).
			Return(nil)

		os.On("Mount", "none", filePath, "tmpfs", uintptr(0), fmt.Sprintf("size=%d", size)).
			Return(nil)

		n, err := volatileDir(name, size, os, os)
		require.NoError(t, err)
		require.Equal(t, n, filePath)
		os.AssertExpectations(t)
	})
}
