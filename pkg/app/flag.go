package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	defaultFlagsDir = "/tmp/flags"
	// LimitedCache represent the flag cache couldn't mount on ssd or hdd
	LimitedCache = "limited-cache"
	// ReadonlyCache represents the flag for cache is read only
	ReadonlyCache = "readonly-cache"
	// NotReachable represents the flag when a grid service is not reachable
	NotReachable = "not-reachable"
)

// SetFlag is used when the /var/cache cannot be mounted on a SSD or HDD,
// it will mount the cache disk on a temporary file system in memory.
func SetFlag(key string) error {
	return setFlag(key, defaultFlagsDir, pkg.DefaultSystemOS)
}

func setFlag(key, flagsDir string, fs pkg.SystemOS) error {
	// creating a file that will be used as a flag
	// the flag can 'warn' other deamons that the /var/cache is not on HDD or SDD
	if err := fs.MkdirAll(flagsDir, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to create flags directory")
	}

	f, err := fs.Create(filepath.Join(flagsDir, key))
	if err != nil {
		return errors.Wrap(err, "failed to create the flag file")
	}

	return f.Close()
}

// CheckFlag checks the status of a flag based on a key
func CheckFlag(key string) bool {
	return checkFlag(key, defaultFlagsDir, pkg.DefaultSystemOS)
}

func checkFlag(key, flagsDir string, os pkg.SystemOS) bool {
	_, err := os.Stat(filepath.Join(flagsDir, key))
	return !os.IsNotExist(err)
}

// DeleteFlag deletes (unsets) a given flag based on a key
func DeleteFlag(key string) error {
	return deleteFlag(key, defaultFlagsDir, pkg.DefaultSystemOS)
}

func deleteFlag(key, flagsDir string, os pkg.SystemOS) error {
	// to avoid "path injection"
	path := filepath.Join(flagsDir, key)
	if filepath.Dir(path) != flagsDir {
		return fmt.Errorf("trying to delete a directory outside of the flags boundaries")
	}

	if err := os.RemoveAll(filepath.Join(flagsDir, key)); err != nil {
		return errors.Wrap(err, "failed to remove the flag file")
	}
	return nil
}
