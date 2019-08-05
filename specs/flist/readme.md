# flist module

This module is responsible to "mount an flist" in the filesystem. The mounted directory contains all the files required by containers or VMs.

The flist module interface is very simple. It does not expose any way to choose where to mount the flist or have any reference to containers or VM. The only functionality is to mount a given flist and received the location where it is mounted.

```go
type FlistModule interface {
    // Mount mounts an flist located at url using the 0-db located at storage.
    // Returns the path in the filesystem where the flist is mounted or an error
    Mount(url string, storage string) (path string, err error)

    // Umount the flist mounted at path
    Umount(path string) error
}
```