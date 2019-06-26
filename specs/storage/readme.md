# Storage module

This module responsible to manage everything related with storage. In 0-OS we have 2 different storage primitives, storage pool and [0-db](https://github.com/threefoldtech/0-db).

Storage pool are used when a direct disk access is required. Typical example would be a container needs to persist some data on disk.
A storage pool is an abstract concept, it just represent something that has a path and on which we can write files.
Usually this is implement using some kind of filesystem on top of one or multiple disks.

[0-DB](https://github.com/threefoldtech/0-db) is the other storage primitives, it provide an efficient Key-Value store interface on top of a disk. 0-db implements different running mode that can server different use cases.

The storage module itself is spitted into 3 sub module, each one responsible for a specific tasks.

List of sub-modules:

- [disks](#disk-sub-module)
- [storage pools](#storage-pool-sub-module)
- [0-db](#0-db-sub-module)
- [booting](#booting)


## Disk sub-module

Responsible to discover and prepare all the disk available on a node to be ready to use for the other sub-modules

### Interface

```go
//Policy defines the preparation policy (raid level, disk types, etc..)
type Policy struct {
    //TODO: to be defines
}

type DiskModule interface {
    // Apply the prepare policy to all connected devices
    // Return a list of disk info
    Prepare(policy Policy) (PoolInfo[],error)

    // Will wipe completely all the content of a disk
    // this is done by overwriting the partition table with random data
    Wipe(pool string) (error)

    // Starts a process that will monitor if some disks are added/removed during the
    // lifetime of the node. This can be used by a higher layer to propagate event up the stack
    // if some application needs to be made aware of disks being added/removed
    Monitor(ctx context.Context) (<-PoolEvent)

    // TODO:  to be be further defined, but we will need something that can give us the exact amount of storage
    // a node has. This is used to compute the resource unit provided by the node.
    Stats() (diskStats, error)
}
```

## Storage pool sub-module

### Interface
Responsible to do the capacity planning of the storage pools on top of the disk prepare by the disk sub-module

> This interface is just for clarification, please check the actual exposed interface under the module code

```go
type Space struct {
    ID string
    Path string
    Size string
}

type StoragePoolModule interface {
    // Reserve creates a storage space that can be used to mount
    // as a volume into a container/VM
    // Reserve is responsible to choose the best suited storage pool to use
    Reserve(diskType DiskType, size int64) (Space, error)
    // Release releases a storage pool acquired by Reserve
    // after this the storage pool need to be considered null and should not be used anymore.
    Release(spaceID string) (error)
}
```

## 0-db sub-module
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

## Booting
When the module boots:
- Make sure to mount all available pools
- Make sure to mount the cache subvolume under the /var/cache
- If no pools exist, it find the first available SSD disk (or hdd if no ssd exists) and create a pool
  with a single disk
  - a cache sub-volume is created and mounted under /var/cache

### Zinit config
The zinit unit file of the module must make sure to provide a test command, that only pass
after the cache volume is mounted 

```yaml
exec: module binary
test: mountpoint /var/cache
```