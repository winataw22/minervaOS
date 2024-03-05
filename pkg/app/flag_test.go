package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

// TestSetFlag tests the setFlag function against multiple scenarios.
// it tests both scenarios of failure/success in setting the flag
func TestSetFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("setFlag invalid flag", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", flagsDir, fs.ModePerm).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&readCloser{}, fmt.Errorf("failed to create file"))

		err := setFlag(testFile, flagsDir, os)
		require.Error(t, err)
		os.AssertExpectations(t)
	})

	t.Run("setFlag valid flag", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", flagsDir, fs.ModePerm).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&readCloser{}, nil)

		err := setFlag(testFile, flagsDir, os)
		require.NoError(t, err)
		os.AssertExpectations(t)
	})
}

// TestCheckFlag tests the checkFlag function against multiple scenarios.
// it tests both scenarios of flag exist/not exist
func TestCheckFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("checkFlag flag exist", func(t *testing.T) {
		os := &pkg.SystemOSMock{}

		os.On("Stat", testFilePath).
			Return(fileInfo{}, nil)

		os.On("IsNotExist", nil).
			Return(false)

		flagExists := checkFlag(testFile, flagsDir, os)
		require.True(t, flagExists)
		os.AssertExpectations(t)
	})

	t.Run("checkFlag flag does not exist", func(t *testing.T) {
		os := &pkg.SystemOSMock{}

		os.On("Stat", testFilePath).
			Return(fileInfo{}, fs.ErrNotExist)

		os.On("IsNotExist", fs.ErrNotExist).
			Return(true)

		flagExists := checkFlag(testFile, flagsDir, os)
		require.False(t, flagExists)
		os.AssertExpectations(t)
	})
}

// TestDeleteFlag tests the deleteFlag function against multiple scenarios.
// it tests both scenarios of valid/invalid cache type
func TestDeleteFlag(t *testing.T) {
	t.Run("delete invalid cache type", func(t *testing.T) {
		key := "/"
		flagsDir := "tmp/flags"

		os := &pkg.SystemOSMock{}
		err := deleteFlag(key, flagsDir, os)
		require.Error(t, err)
		os.AssertExpectations(t)
	})

	t.Run("delete valid cache type", func(t *testing.T) {
		key := LimitedCache
		flagsDir := "tmp/flags"

		os := &pkg.SystemOSMock{}
		os.On("RemoveAll", filepath.Join(flagsDir, key)).
			Return(nil)

		err := deleteFlag(key, flagsDir, os)
		require.NoError(t, err)
		os.AssertExpectations(t)
	})
}
