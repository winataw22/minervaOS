package pkg

import (
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs

//go:generate zbusc -module flist -version 0.0.1 -name flist -package stubs github.com/threefoldtech/zos/pkg+Flister stubs/flist_stub.go

var (
	//DefaultMountOptions has sane values for mount
	DefaultMountOptions = MountOptions{
		ReadOnly: false,
		Limit:    256, //Mib
		Type:     zos.SSDDevice,
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
	// Limit size of read-write layer in Mib
	Limit uint64
	// Type of disk to use
	Type zos.DeviceType
}

//Flister is the interface for the flist module
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage.
	// MountOptions, can be nil, in that case falls to default, other wise
	// use the provided values.
	// Returns the path in the filesystem where the flist is mounted or an error
	Mount(url string, storage string, opts MountOptions) (path string, err error)

	// Mount mounts an flist located at url using the 0-db located at storage.
	// MountOptions, can be nil, in that case falls to default, other wise
	// use the provided values.
	// Name is a unique name to identify this mount. The flist can later be unmounted
	// with the same name. It is up to the caller to ensure `name` is unique.
	// Returns the path in the filesystem where the flist is mounted or an error
	NamedMount(name string, url string, storage string, ots MountOptions) (path string, err error)

	// Umount the flist mounted at path
	Umount(path string) error

	// NamedUmount unmounts the flist mounted via the NamedMount call, with the same name
	NamedUmount(path string) error

	// HashFromRootPath returns flist hash from a running g8ufs mounted with NamedMount
	HashFromRootPath(name string) (string, error)

	// FlistHash returns md5 of flist if available (requesting the hub)
	FlistHash(url string) (string, error)
}
