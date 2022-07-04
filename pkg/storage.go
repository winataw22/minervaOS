package pkg

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module storage -version 0.0.1 -name storage -package stubs github.com/threefoldtech/zos/pkg+StorageModule stubs/storage_stub.go

// RaidProfile type

// ErrNotEnoughSpace indicates that there is not enough space in a pool
// of the requested type to create the filesystem
type ErrNotEnoughSpace struct {
	DeviceType DeviceType
}

func (e ErrNotEnoughSpace) Error() string {
	return fmt.Sprintf("not enough space left in pools of this type %s", e.DeviceType)
}

// ErrInvalidDeviceType raised when trying to allocate space on unsupported device type
type ErrInvalidDeviceType struct {
	DeviceType DeviceType
}

func (e ErrInvalidDeviceType) Error() string {
	return fmt.Sprintf("invalid device type '%s'. type unknown", e.DeviceType)
}

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType = zos.DeviceType

type (
	// BrokenDevice is a disk which is somehow not fully functional. Storage keeps
	// track of disks which have failed at some point, so they are not used, and
	// to be able to later report this to other daemons.
	BrokenDevice struct {
		// Path to allow identification of the disk
		Path string
		// Err returned which lead to the disk being marked as faulty
		Err error
	}

	// BrokenPool contains info about a malfunctioning storage pool
	BrokenPool struct {
		// Label of the broken pool
		Label string
		// Err returned by the action which let to the pool being marked as broken
		Err error
	}
)

// Usage struct
type Usage struct {
	Size gridtypes.Unit
	Used gridtypes.Unit
}

// Volume struct is a btrfs subvolume
type Volume struct {
	Name  string
	Path  string
	Usage Usage
}

// Device struct is a full hdd
type Device struct {
	Path  string
	ID    string
	Usage Usage
}

// StorageModule is the storage subsystem interface
// this should allow you to work with the following types of storage medium
// - full disks (device) (these are used by zdb)
// - subvolumes these are used as a read-write layers for 0-fs mounts
// - vdisks are used by zmachines
// this works as following:
// a storage module maintains a list of ALL disks on the system
// separated in 2 sets of pools (SSDs, and HDDs)
// ssd pools can only be used for
// - subvolumes
// - vdisks
// hdd pools are only used by zdb as one disk
type StorageModule interface {
	// Cache method return information about zos cache volume
	Cache() (Volume, error)

	// Total gives the total amount of storage available for a device type
	Total(kind DeviceType) (uint64, error)
	// BrokenPools lists the broken storage pools that have been detected
	BrokenPools() []BrokenPool
	// BrokenDevices lists the broken devices that have been detected
	BrokenDevices() []BrokenDevice
	//Monitor returns stats stream about pools
	Monitor(ctx context.Context) <-chan PoolsStats

	// Volume management

	// VolumeCreate creates a new volume
	VolumeCreate(name string, size gridtypes.Unit) (Volume, error)

	// VolumeUpdate updates the size of an existing volume
	VolumeUpdate(name string, size gridtypes.Unit) error

	// VolumeLookup return volume information for given name
	VolumeLookup(name string) (Volume, error)

	// VolumeDelete deletes a volume by name
	VolumeDelete(name string) error

	// VolumeList list all volumes
	VolumeList() ([]Volume, error)

	// Virtual disk management

	// DiskCreate creates a virtual disk given name and size
	DiskCreate(name string, size gridtypes.Unit) (VDisk, error)

	// DiskResize resizes the disk to given size
	DiskResize(name string, size gridtypes.Unit) (VDisk, error)

	// DiskWrite writes the given raw image to disk
	DiskWrite(name string, image string) error

	// DiskFormat makes sure disk has filesystem, if it already formatted nothing happens
	DiskFormat(name string) error

	// DiskLookup looks up vdisk by name
	DiskLookup(name string) (VDisk, error)

	// DiskExists checks if disk exists
	DiskExists(name string) bool

	// DiskDelete deletes a disk
	DiskDelete(name string) error

	DiskList() ([]VDisk, error)
	// Device management

	//Devices list all "allocated" devices
	Devices() ([]Device, error)

	// DeviceAllocate allocates a new device (formats and give a new ID)
	DeviceAllocate(min gridtypes.Unit) (Device, error)

	// DeviceLookup inspects a previously allocated device
	DeviceLookup(name string) (Device, error)
}

// VDisk info returned by a call to inspect
type VDisk struct {
	// Path to disk
	Path string
	// Size in bytes
	Size int64
}

// Name returns the Name part of the disk path
func (d *VDisk) Name() string {
	return filepath.Base(d.Path)
}
