package app

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// bootedPath is the path where to store the booted flag
var bootedPath = "/var/run/modules"

// MarkBooted creates a file in a memory
// this file then can be used to check if "something" has been restared
// if its the first time it starts
func MarkBooted(name string) error {
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
	path := filepath.Join(bootedPath, name)
	_, err := os.Stat(path)
	return err != nil
}
