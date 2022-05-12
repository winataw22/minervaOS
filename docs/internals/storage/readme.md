# Storage Module

## ZBus

Storage module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| storage|[storage](#interface)| 0.0.1|

## Introduction

This module responsible to manage everything related with storage. On start, storaged holds ownership of all node disks, and it separate it into 2 different sets:

- SSD Storage: For each ssd disk available, a storage pool of type SSD is created
- HDD Storage: For each HDD disk available, a storage pool of type HDD is created

Then `storaged` can provide the following storage primitives:
- `subvolume`: (with quota). The btrfs subvolume can be used by used by `flistd` to support read-write operations on flists. Hence it can be used as rootfs for containers and VMs. This storage primitive is only supported on `ssd` pools.
    - On boot, storaged will always create a permanent subvolume with id `zos-cache` (of 100G) which will be used by the system to persist state and to hold cache of downloaded files.
- `vdisk`: Virtual disk that can be attached to virtual machines. this is only possible on `ssd` pools.
- `device`: that is a full disk that gets allocated and used by a single `0-db` service. Note that a single 0-db instance can serve multiple zdb namespaces for multiple users. This is only possible for on `hdd` pools.

You already can tell that ZOS can work fine with no HDD (it will not be able to server zdb workloads though), but not without SSD. Hence a zos with no SSD will never register on the grid.

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

### Interface

```go

// StorageModule is the storage subsystem interface
// this should allow you to work with the following types of storage medium
// - full disks (device) (these are used by zdb)
// - subvolumes these are used as a read-write layers for 0-fs mounts
// - vdisks are used by zmachines
// this works as following:
// a storage module maintains a list of ALL disks on the system
// separated in 2 sets of pools (SSDs, and HDDs)
// ssd pools can only be used for
// - subvolumes
// - vdisks
// hdd pools are only used by zdb as one disk
type StorageModule interface {
	// Cache method return information about zos cache volume
	Cache() (Volume, error)

	// Total gives the total amount of storage available for a device type
	Total(kind DeviceType) (uint64, error)
	// BrokenPools lists the broken storage pools that have been detected
	BrokenPools() []BrokenPool
	// BrokenDevices lists the broken devices that have been detected
	BrokenDevices() []BrokenDevice
	//Monitor returns stats stream about pools
	Monitor(ctx context.Context) <-chan PoolsStats

	// Volume management

	// VolumeCreate creates a new volume
	VolumeCreate(name string, size gridtypes.Unit) (Volume, error)

	// VolumeUpdate updates the size of an existing volume
	VolumeUpdate(name string, size gridtypes.Unit) error

	// VolumeLookup return volume information for given name
	VolumeLookup(name string) (Volume, error)

	// VolumeDelete deletes a volume by name
	VolumeDelete(name string) error

	// VolumeList list all volumes
	VolumeList() ([]Volume, error)

	// Virtual disk management

	// DiskCreate creates a virtual disk given name and size
	DiskCreate(name string, size gridtypes.Unit) (VDisk, error)

	// DiskResize resizes the disk to given size
	DiskResize(name string, size gridtypes.Unit) (VDisk, error)

	// DiskWrite writes the given raw image to disk
	DiskWrite(name string, image string) error

	// DiskFormat makes sure disk has filesystem, if it already formatted nothing happens
	DiskFormat(name string) error

	// DiskLookup looks up vdisk by name
	DiskLookup(name string) (VDisk, error)

	// DiskExists checks if disk exists
	DiskExists(name string) bool

	// DiskDelete deletes a disk
	DiskDelete(name string) error

	DiskList() ([]VDisk, error)
	// Device management

	//Devices list all "allocated" devices
	Devices() ([]Device, error)

	// DeviceAllocate allocates a new device (formats and give a new ID)
	DeviceAllocate(min gridtypes.Unit) (Device, error)

	// DeviceLookup inspects a previously allocated device
	DeviceLookup(name string) (Device, error)
}
```
