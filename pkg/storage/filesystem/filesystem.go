package filesystem

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Usage struct (in bytes)
type Usage struct {
	// Size is allocated space for this Volume
	// if 0 it means it has no limit.
	// if it has no-limit, the Used attribute
	// will be the total size of actual files
	// inside the volume. It also means the Used
	// can keep growing to the max possible which
	// is the size of the pool
	Size uint64
	// Used can be one of 2 things:
	// - If Size is not zero (so size is limited), Used will always equal to size
	//   because that's the total reserved space for that volume.
	// - If Size is zero, (no limit) Used will be the total actual size of all
	//   files in that volume.
	// The reason Used is done this way, it will make it easier to compute
	// all allocated space in a pool by going over all volumes and add the
	// used on each. It does not matter if this space is reserved but not used
	// because it means we can't allocate over that.
	//
	// NOTE: Special case, if this is a `zdb` volume the Used is instead
	// have the total size of reserved namespaces in that volume
	Used uint64

	// In case of `limited` volume (with quota) Excl will have a "guessed"
	// value of the total used space by files. This value is not accurate
	// and Used should be used instead for all capacity planning.
	Excl uint64
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
	// Volumes are all subvolumes of this volume
	Volumes() ([]Volume, error)
	// AddVolume adds a new subvolume with the given name
	AddVolume(name string) (Volume, error)
	// RemoveVolume removes a subvolume with the given name
	RemoveVolume(name string) error
	// Shutdown spins down the device where the pool is mounted
	Shutdown() error
	// Device return device associated with pool
	Device() DeviceInfo
	// SetType sets a device type on the pool. this will make
	// sure that the detected device type is reported
	// correctly by calling the Type() method.
	SetType(typ zos.DeviceType) error
	// Type returns the device type set by a previous call
	// to SetType.
	Type() (zos.DeviceType, bool, error)
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
