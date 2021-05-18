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
	Mounted() (string, bool)
	// Mount the pool, the mountpoint is returned
	Mount() (string, error)
	// MountWithoutScan the pool, the mountpoint is returned.
	// Does not scan for btrfs
	MountWithoutScan() (string, error)
	// UnMount the pool
	UnMount() error
	//AddDevice to the pool
	AddDevice(device *Device) error
	// RemoveDevice from the pool
	RemoveDevice(device *Device) error
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
	// Devices list attached devices
	Devices() []*Device

	// Shutdown spins down the device where the pool is mounted
	Shutdown() error
}

// Filter closure for Filesystem list
type Filter func(pool Pool) bool

// All is default filter
func All(Pool) bool {
	return true
}

// Filesystem defines a filesystem interface
type Filesystem interface {
	// Create a new filesystem.
	//
	// name: name of the filesystem
	// devices: list of devices to use in the filesystem
	// profile: Raid profile of the filesystem
	Create(ctx context.Context, name string, profile pkg.RaidProfile, devices ...*Device) (Pool, error)

	// CreateForce creates a new filesystem with force
	// It will delete existing data and partition tables
	CreateForce(ctx context.Context, name string, profile pkg.RaidProfile, devices ...*Device) (Pool, error)
	// List all existing filesystems on the node
	List(ctx context.Context, filter Filter) ([]Pool, error)
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
