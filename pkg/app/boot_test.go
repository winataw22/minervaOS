package app

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

type readCloser struct {
	io.Reader
}

func (fs *readCloser) Close() error {
	return nil
}

type fileInfo struct{}

func (f fileInfo) Name() string {
	return ""
}

func (f fileInfo) Size() int64 {
	return 0
}

func (f fileInfo) Mode() fs.FileMode {
	return fs.FileMode(0000)
}

func (f fileInfo) ModTime() time.Time {
	return time.Now()
}

func (f fileInfo) IsDir() bool {
	return false
}

func (f fileInfo) Sys() any {
	return nil
}

// TestMarkBooted tests the markBooted function against multiple scenarios.
// it tests different scenarios of failures during directory creation or file creation.
func TestMarkBooted(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("markBooted failed to MkdirAll", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		errMkdir := fmt.Errorf("failed to MkdirAll")
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(fmt.Errorf("failed to MkdirAll"))

		err := markBooted(testFile, bootedPath, os)
		require.Error(t, err)
		require.Equal(t, err, errMkdir)
		os.AssertExpectations(t)
	})

	t.Run("markBooted failed to Create", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&readCloser{}, fmt.Errorf("couldn't create file with name file1"))

		err := markBooted(testFile, bootedPath, os)
		require.Error(t, err)
		os.AssertExpectations(t)
	})

	t.Run("markBooted valid file", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&readCloser{}, nil)

		err := markBooted(testFile, bootedPath, os)
		require.NoError(t, err)
		os.AssertExpectations(t)
	})
}

// TestIsFirstBoo tests the isFirstBoot function against multiple scenarios.
// it tests both scenarios of the file being first booted or not
func TestIsFirstBoot(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("the file is first booted", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("Stat", testFilePath).
			Return(fileInfo{}, nil)

		firstBoot := isFirstBoot(testFile, bootedPath, os)
		require.False(t, firstBoot)
		os.AssertExpectations(t)
	})

	t.Run("the file is not first booted", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("Stat", testFilePath).
			Return(fileInfo{}, fmt.Errorf("couldn't find file in the bootedPath"))

		firstBoot := isFirstBoot(testFile, bootedPath, os)
		require.True(t, firstBoot)
		os.AssertExpectations(t)
	})
}
