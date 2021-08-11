package filesystem

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

// Usage struct (in bytes)
type Usage struct {
	Size uint64
	Used uint64
}

// Volume represents a logical volume in the pool. Volumes can be nested
type Volume interface {
	// Volume ID
	ID() int
	// Path of the volume
	Path() string
	// Usage reports the current usage of the volume
	Usage() (Usage, error)
	// Limit the maximum size of the volume
	Limit(size uint64) error
	// Name of the volume
	Name() string
	// FsType of the volume
	FsType() string
}

// Pool represents a created filesystem
type Pool interface {
	Volume
	// Mounted returns whether the pool is mounted or not. If it is mounted,
	// the mountpoint is returned
	Mounted() (string, error)
	// Mount the pool, the mountpoint is returned
	Mount() (string, error)
	// UnMount the pool
	UnMount() error
	//AddDevice to the pool
	// RemoveDevice from the pool
	// Type of the physical storage in this pool
	Type() pkg.DeviceType
	// Reserved is reserved size of the devices in bytes
	Reserved() (uint64, error)
	// Volumes are all subvolumes of this volume
	Volumes() ([]Volume, error)
	// AddVolume adds a new subvolume with the given name
	AddVolume(name string) (Volume, error)
	// RemoveVolume removes a subvolume with the given name
	RemoveVolume(name string) error
	// Shutdown spins down the device where the pool is mounted
	Shutdown() error
	// Device return device associated with pool
	Device() Device
}

// Filter closure for Filesystem list
type Filter func(pool Pool) bool

// All is default filter
func All(Pool) bool {
	return true
}

// Partprobe runs partprobe
func Partprobe(ctx context.Context) error {
	if _, err := run(ctx, "partprobe"); err != nil {
		return errors.Wrap(err, "partprobe failed")
	}
	if _, err := run(ctx, "udevadm", "settle"); err != nil {
		return errors.Wrap(err, "failed to wait for udev settle")
	}
	return nil
}
