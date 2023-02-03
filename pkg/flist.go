package pkg

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

//go:generate mkdir -p stubs

//go:generate zbusc -module flist -version 0.0.1 -name flist -package stubs github.com/threefoldtech/zos/pkg+Flister stubs/flist_stub.go

var (
	//DefaultMountOptions has sane values for mount
	DefaultMountOptions = MountOptions{
		ReadOnly: false,
		Limit:    256 * gridtypes.Megabyte, //Mib
	}

	//ReadOnlyMountOptions shortcut for readonly mount options
	ReadOnlyMountOptions = MountOptions{
		ReadOnly: true,
	}
)

// MountOptions struct
type MountOptions struct {
	// ReadOnly
	ReadOnly bool
	// Limit size of read-write layer
	Limit gridtypes.Unit
	// optional storage url (default to hub storage)
	Storage string
	// PersistedVolume used in RW mode. If not provided
	// one that will be created automatically with `Limit` that uses the same mount
	// name, and will be delete (by name) on Unmount. If provided, make sure
	// use use a different name than the mount id, or it will also get deleted
	// on unmount.
	PersistedVolume string
}

// Flister is the interface for the flist module
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage
	// in a RO mode. note that there is no way u can unmount a ro flist because
	// it can be shared by many users, it's then up to system to decide if the
	// mount is not needed anymore and clean it up
	Mount(name, url string, opt MountOptions) (path string, err error)

	// UpdateMountSize change the mount size
	UpdateMountSize(name string, limit gridtypes.Unit) (path string, err error)

	// Umount a RW mount. this only unmounts the RW layer and remove the assigned
	// volume.
	Unmount(name string) error

	// HashFromRootPath returns flist hash from a running g8ufs mounted with NamedMount
	HashFromRootPath(name string) (string, error)

	// FlistHash returns md5 of flist if available (requesting the hub)
	FlistHash(url string) (string, error)

	Exists(name string) (bool, error)
}
