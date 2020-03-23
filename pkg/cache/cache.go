package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/threefoldtech/zos/pkg/storage"
)

// ForeverDir creates a new cache directory that is stored on the persistent cache.
// This means data stored in this directory will survive reboot
// It is the caller's responsibility to remove the directory when no longer needed.
func ForeverDir(name string) (string, error) {
	name = filepath.Join(storage.CacheTarget, "cache", name)
	return name, os.MkdirAll(name, 0700)
}

// VolatileDir creates a new cache directory that is stored on a tmpfs.
// This means data stored in this directory will NOT survive a reboot.
// Use this when you need to store data that needs to survice deamon reboot but not between reboots
// It is the caller's responsibility to remove the directory when no longer needed.
// If the directory already exist, the the function does nothing and return successfully
func VolatileDir(name string, size uint64) (string, error) {
	const volatileBaseDir = "/var/run/cache"
	name = filepath.Join(volatileBaseDir, name)
	if err := os.MkdirAll(volatileBaseDir, 0700); err != nil {
		return name, err
	}

	if err := os.Mkdir(name, 0700); err != nil {
		return name, err
	}
	return name, syscall.Mount("", name, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}
