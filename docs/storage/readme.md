# Storage Module

## ZBus 
Storage module is available on zbus over the following channel

| module | object | version | 
|--------|--------|---------|
| storage|[storage](#disk-object)| 0.0.1|

## Introduction
This module responsible to manage everything related with storage. In 0-OS we have 2 different storage primitives, storage pool and [0-db](https://github.com/threefoldtech/0-db).

Storage pool are used when a direct disk access is required. Typical example would be a container needs to persist some data on disk.
A storage pool is an abstract concept, it just represent something that has a path and on which we can write files.
Usually this is implement using some kind of filesystem on top of one or multiple disks.

[0-DB](https://github.com/threefoldtech/0-db) is the other storage primitives, it provide an efficient Key-Value store interface on top of a disk. 0-db implements different running mode that can server different use cases.

The storage module itself is spitted into 3 sub module, each one responsible for a specific tasks.

List of sub-modules:

- [disks](#disk-sub-module)
- [0-db](#0-db-sub-module)
- [booting](#booting)

## On Node Booting
When the module boots:
- Make sure to mount all available pools
- Scan available disks that are not used by any pool and create new pools on those disks. (all pools now are created with `RaidSingle` policy)
- Try to find and mount a cache sub-volume under /var/cache.
- If no cache sub-volume is available a new one is created and then mounted.

### zinit unit
The zinit unit file of the module specify the command line,  test command, and the order where the services need to be booted.

Storage module is a dependency for almost all other system modules, hence it has high boot presidency (calculated on boot) by zinit based on the configuration.

The storage module is only considered running, if (and only if) the /var/cache is ready

```yaml
exec: storaged
test: mountpoint /var/cache
```

## Disk object

Responsible to discover and prepare all the disk available on a node to be ready to use for the other sub-modules

### Interface

```go
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

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice = "SSD"
	HDDDevice = "HDD"
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

// StorageModule defines the api for storage
type StorageModule interface {
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

```

## 0-db object
> This object is `NOT IMPLEMENTED YET`

Responsible to do the capacity planning of the 0-db on top of the disk prepare by the disk sub-module

### Interface

```go
type ZDBNamespace struct {
    ID string
    DiskType DiskType
    Size int64
    Mode ZdbMode
    Password string
    Port int // Listen port of the 0-db owning the namespace
}

type ZDBModule interface {
    // ReserveNamespace reserve a 0-db namespace of specified size, type and mode
    // ReserveNamespace is responsible for the capacity planning and must decide what is the
    // best place where to create this namespace. If can happen that ReserveNamespace endup
    // deploying a new 0-db to be able to create the namespace
    ReserveNamespace(diskType DiskType, size int64, mode ZdbMode, password string) (ZDBNamespace, error)

    // ReleaseNamespace delete a 0-db namespace acquired by ReserveNamespace
    ReleaseNamespace(nsID string) (error)
}
```
