package app

import (
	"os"
	"path/filepath"
)

const bootedPath = "/var/run/modules"

// MarkBooted creates a file in a memory
// this file then can be used to check if "something" has been restared
// if its the first time it starts
func MarkBooted(name string) error {
	if err := os.MkdirAll(bootedPath, 0770); err != nil {
		return err
	}
	path := filepath.Join(bootedPath, name)
	_, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0666)
	return err
}

// IsFirstBoot checks if the a file has been created by MarkBooted function
func IsFirstBoot(name string) bool {
	path := filepath.Join(bootedPath, name)
	_, err := os.Stat(path)
	return !(err == nil)
}
