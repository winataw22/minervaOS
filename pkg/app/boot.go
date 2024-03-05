package app

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

// defaultBootedPath is the path where to store the booted flag
const defaultBootedPath = "/var/run/modules"

// MarkBooted creates a file in a memory
// this file then can be used to check if "something" has been restared
// if its the first time it starts
func MarkBooted(name string) error {
	return markBooted(name, defaultBootedPath, pkg.DefaultSystemOS)
}

func markBooted(name string, bootedPath string, os pkg.SystemOS) error {
	if err := os.MkdirAll(bootedPath, 0770); err != nil {
		return err
	}
	path := filepath.Join(bootedPath, name)
	marker, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to mark service as booted: %s", name)
	}

	return marker.Close()
}

// IsFirstBoot checks if the a file has been created by MarkBooted function
func IsFirstBoot(name string) bool {
	return isFirstBoot(name, defaultBootedPath, pkg.DefaultSystemOS)
}

func isFirstBoot(name string, bootedPath string, os pkg.SystemOS) bool {
	path := filepath.Join(bootedPath, name)
	_, err := os.Stat(path)
	return err != nil
}
