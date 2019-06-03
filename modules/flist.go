package modules

//go:generate mkdir -p stubs

//go:generate zbusc -module flist -version 0.0.1 -name flist -package stubs github.com/threefoldtech/zosv2/modules+Flister stubs/flist_stub.go

//Flister is the interface for the flist module
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage.
	// Returns the path in the filesystem where the flist is mounted or an error
	Mount(url string, storage string) (path string, err error)

	// Umount the flist mounted at path
	Umount(path string) error
}
