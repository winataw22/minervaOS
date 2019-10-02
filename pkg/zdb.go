package pkg

//go:generate mkdir -p stubs
//go:generate zbusc -module storage -version 0.0.1 -name storage -package stubs github.com/threefoldtech/zos/pkg+ZDBAllocater stubs/zdb_stub.go

// ZDBMode is the enumeration of the modes 0-db can operate in
type ZDBMode string

// Enumeration of the modes 0-db can operate in
const (
	ZDBModeUser = "user"
	ZDBModeSeq  = "seq"
)

// ZDBNamespace is a 0-db namespace
type ZDBNamespace struct {
	ID       string
	DiskType DeviceType
	Size     uint64
	Mode     ZDBMode
	Password string
	Port     int // Listen port of the 0-db owning the namespace
}

// ZDBAllocater is the zbus interface of the storage module responsible
// for 0-db allocation
type ZDBAllocater interface {
	// Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
	// of specified size, type and mode
	// it returns the volume ID and its path or an error if it couldn't allocate enough storage
	Allocate(diskType DeviceType, size uint64, mode ZDBMode) (string, string, error)

	// Claim let the system claim the allocated storage used by a 0-db namespace
	Claim(ID string, size uint64) error
}
