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
//go:generate zbusc -module storage -version 0.0.1 -name vdisk -package stubs github.com/threefoldtech/zos/pkg+VDiskModule stubs/vdisk_stub.go

// RaidProfile type
type RaidProfile string

const (
	// Single profile
	Single RaidProfile = "single"
	// Raid0 profile
	Raid0 RaidProfile = "raid0"
	// Raid1 profile
	Raid1 RaidProfile = "raid1"
	// Raid10 profile
	Raid10 RaidProfile = "raid10"
)

// ErrNotEnoughSpace indicates that there is not enough space in a pool
// of the requested type to create the filesystem
type ErrNotEnoughSpace struct {
	DeviceType DeviceType
}

func (e ErrNotEnoughSpace) Error() string {
	return fmt.Sprintf("Not enough space left in pools of this type %s", e.DeviceType)
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

// Validate make sure profile is correct
func (p RaidProfile) Validate() error {
	if _, ok := raidProfiles[p]; !ok {
		return fmt.Errorf("not supported raid profile '%s'", p)
	}

	return nil
}

var (
	raidProfiles = map[RaidProfile]struct{}{
		Single: {}, Raid1: {}, Raid10: {},
	}
	// DefaultPolicy value
	DefaultPolicy = StoragePolicy{
		Raid: Single,
	}

	// NullPolicy does not create pools
	NullPolicy = StoragePolicy{}
)

// StoragePolicy describes the pool creation policy
type StoragePolicy struct {
	// Raid profile for this policy
	Raid RaidProfile
	// Number of disks to use in a single pool
	// note that, the disks count must be valid for
	// the chosen raid profile.
	Disks uint8

	// Only create this amount of storage pools. Default to 0 -> unlimited.
	// The spared disks can later be used in automatic repair if a physical
	// disk got corrupt or bad.
	// Note that if it's set to 0 (unlimited), some disks might be spared anyway
	// in case the number of disks required in the policy doesn't add up to pools
	// for example, a pool of 2s on a machine with 5 disks.
	MaxPools uint8
}

// Usage struct
type Usage struct {
	Size gridtypes.Unit
	Used gridtypes.Unit
}

// Filesystem represents a storage space that can be used as a filesystem
type Filesystem struct {
	// Filesystem ID
	ID int
	// Path of the Filesystem
	Path string
	// Usage reports the current usage of the Filesystem
	Usage Usage
	// Name of the Filesystem
	Name string
	// FsType of the Filesystem
	FsType string
	// DiskType of the Filesystem
	DiskType DeviceType
}

// VolumeAllocater is the zbus interface of the storage module responsible
// for volume allocation
type VolumeAllocater interface {
	// CreateFilesystem creates a filesystem with a given size. The filesystem
	// is mounted, and the path to the mountpoint is returned. The filesystem
	// is only attempted to be created in a pool of the given type. If no
	// more space is available in such a pool, `ErrNotEnoughSpace` is returned.
	// It is up to the caller to handle such a situation and decide if he wants
	// to try again on a different devicetype
	CreateFilesystem(name string, size gridtypes.Unit, poolType DeviceType) (Filesystem, error)

	// UpdateFilesystem changes a filesystem size to given value.
	UpdateFilesystem(name string, size gridtypes.Unit) (Filesystem, error)
	// ReleaseFilesystem signals that the named filesystem is no longer needed.
	// The filesystem will be unmounted and subsequently removed.
	// All data contained in the filesystem will be lost, and the
	// space which has been reserved for this filesystem will be reclaimed.
	ReleaseFilesystem(name string) error

	// ListFilesystems return all the filesystem managed by storeaged present on the nodes
	// this can be an expensive call on server with a lot of disk, don't use it in a
	// intensive loop
	// Special filesystem like internal cache and vdisk are not return by this function
	// to access them use the GetCacheFS or GetVdiskFS
	ListFilesystems() ([]Filesystem, error)

	// Path return the filesystem named name
	// if no filesystem with this name exists, an error is returned
	Path(name string) (Filesystem, error)

	// GetCacheFS return the special filesystem used by 0-OS to store internal state and flist cache
	GetCacheFS() (Filesystem, error)
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

// VDiskModule interface
type VDiskModule interface {
	// AllocateDisk with given id and size, return path to virtual disk
	Allocate(id string, size gridtypes.Unit) (string, error)
	// DeallocateVDisk removes a virtual disk
	Deallocate(id string) error
	// Exists checks if disk with that ID already allocated
	Exists(id string) bool
	// Inspect return info about the disk
	Inspect(id string) (VDisk, error)
	// List lists all the available vdisks
	List() ([]VDisk, error)
}

// StorageModule defines the api for storage
type StorageModule interface {
	VolumeAllocater
	ZDBAllocater

	// Total gives the total amount of storage available for a device type
	Total(kind DeviceType) (uint64, error)
	// BrokenPools lists the broken storage pools that have been detected
	BrokenPools() []BrokenPool
	// BrokenDevices lists the broken devices that have been detected
	BrokenDevices() []BrokenDevice

	//Monitor returns stats stream about pools
	Monitor(ctx context.Context) <-chan PoolsStats
}
