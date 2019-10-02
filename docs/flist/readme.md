# Flist module

## Zbus

Flist module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
|flist   |[flist](#public-interface)| 0.0.1

## Home Directory
flist keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/containerd`|

## Introduction

This module is responsible to "mount an flist" in the filesystem of the node. The mounted directory contains all the files required by containers or (in the future) VMs.

The flist module interface is very simple. It does not expose any way to choose where to mount the flist or have any reference to containers or VM. The only functionality is to mount a given flist and received the location where it is mounted. It is up to the above layer to do something useful with this information

The flist module itself doesn't contain the logic to understand the flist format or to run the fuse filesystem. It is just a wrapper that manged [0-fs](https://github.com/threefoldtech/0-fs) processes.

Its only job is to download the flist, prepare the isolation of all the data and then starts 0-fs with the proper arguments

## Public interface [![GoDoc](https://godoc.org/github.com/threefoldtech/zos/pkg/flist?status.svg)](https://godoc.org/github.com/threefoldtech/zos/pkg/flist)

```go
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage.
	// Returns the path in the filesystem where the flist is mounted or an error
	Mount(url string, storage string) (path string, err error)

	// Umount the flist mounted at path
	Umount(path string) error
}
```

## zinit unit

The zinit unit file of the module specify the command line,  test command, and the order where the services need to be booted.

Flist module depends on the storage and network pkg.
This is because he needs connectivity to download flist and data and he needs storage to be able to cache the data once downloaded.

Flist doesn't do anything special on the system except creating a bunch of directory it will use during its lifetime
