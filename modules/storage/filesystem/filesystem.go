package filesystem

import (
	"context"

	"github.com/threefoldtech/zosv2/modules"
)

// Usage struct
type Usage struct {
	Size uint64
	Used uint64
}

// Volume represents a logical volume in the pool. Volumes can be nested
type Volume interface {
	// Path of the volume
	Path() string
	// Volumes are all subvolumes of this volume
	Volumes() ([]Volume, error)
	// AddVolume adds a new subvolume with the given name
	AddVolume(name string) (Volume, error)
	// RemoveVolume removes a subvolume with the given name
	RemoveVolume(name string) error
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
	Mounted() (string, bool)
	// Mount the pool, the mountpoint is returned
	Mount() (string, error)
	// UnMount the pool
	UnMount() error
	//AddDevice to the pool
	AddDevice(device *Device) error
	// RemoveDevice from the pool
	RemoveDevice(device *Device) error
	// Type of the physical storage in this pool
	Type() modules.DeviceType

	// Health() ?
}

// Filesystem defines a filesystem interface
type Filesystem interface {
	// Create a new filesystem.
	//
	// name: name of the filesystem
	// devices: list of devices to use in the filesystem
	// profile: Raid profile of the filesystem
	Create(ctx context.Context, name string, devices DeviceCache, profile modules.RaidProfile) (Pool, error)
	// List all existing filesystems on the node
	List(ctx context.Context) ([]Pool, error)
}
