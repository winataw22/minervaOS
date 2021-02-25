package pkg

import "github.com/threefoldtech/zos/pkg/gridtypes/zos"

//go:generate mkdir -p stubs
//go:generate zbusc -module storage -version 0.0.1 -name storage -package stubs github.com/threefoldtech/zos/pkg+ZDBAllocater stubs/zdb_stub.go

// ZDBMode is the enumeration of the modes 0-db can operate in
type ZDBMode = zos.ZDBMode

// ZDBNamespace is a 0-db namespace
type ZDBNamespace struct {
	ID       string
	DiskType DeviceType
	Size     uint64
	Mode     ZDBMode
	Password string
	Port     int // Listen port of the 0-db owning the namespace
}

// Allocation is returned when calling the ZDB allocate. it contains
// the volume ID and the volume path that has the namespace allocated
type Allocation struct {
	VolumeID   string
	VolumePath string
}

// ZDBAllocater is the zbus interface of the storage module responsible
// for 0-db allocation
type ZDBAllocater interface {
	// Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
	// of specified size, type and mode
	// it returns the volume ID and its path or an error if it couldn't allocate enough storage
	// Note: if allocation already exists with the namespace name, the current allocation is returned
	// so no need to call Find before calling allocate
	Allocate(namespace string, diskType DeviceType, size uint64, mode ZDBMode) (Allocation, error)

	// Find searches the system for the current allocation for the namespace
	// Return error = "not found" if no allocation exists.
	Find(namespace string) (allocation Allocation, err error)
}
