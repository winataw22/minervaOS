package modules

import (
	"errors"
	"fmt"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module storage -version 0.0.1 -name storage -package stubs github.com/threefoldtech/zosv2/modules+StorageModule stubs/storage_stub.go

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

var (
	// ErrNotEnoughSpace indicates that there is not enough space in a pool
	// of the requested type to create the filesystem
	ErrNotEnoughSpace = errors.New("Not enough space left in pools of this type")
)

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice = "SSD"
	HDDDevice = "HDD"
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
		Single: struct{}{}, Raid1: struct{}{}, Raid10: struct{}{},
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

// VolumeAllocater is the zbus interface of the storage module responsible
// for volume allocation
type VolumeAllocater interface {
	// CreateFilesystem creates a filesystem with a given size. The filesystem
	// is mounted, and the path to the mountpoint is returned. The filesystem
	// is only attempted to be created in a pool of the given type. If no
	// more space is available in such a pool, `ErrNotEnoughSpace` is returned.
	// It is up to the caller to handle such a situation and decide if he wants
	// to try again on a different devicetype
	CreateFilesystem(name string, size uint64, poolType DeviceType) (string, error)

	// ReleaseFilesystem signals that the named filesystem is no longer needed.
	// The filesystem will be unmounted and subsequently removed.
	// All data contained in the filesystem will be lost, and the
	// space which has been reserved for this filesystem will be reclaimed.
	ReleaseFilesystem(name string) error

	// Path return the path of the mountpoint of the named filesystem
	// if no volume with name exists, an empty path and an error is returned
	Path(name string) (path string, err error)
}

// StorageModule defines the api for storage
type StorageModule interface {
	VolumeAllocater
	ZDBAllocater
}
