# Storage

Implementation of the storage module interface

## Mountpoints

Every volume which is managed by the storage module is mounted in `/mnt`.
The name of the mountpoint is the label of the volume. For instance, a
volume with label `volumelabel` will be mounted in `/mnt/volumelabel`. The volumes
are mounted when the storage module is started, and remain mounted after that.
In case the storae module exits in any way, there is no attempt to unmount
any volume.

Since filesystems are created as subvolumes within the "root" volumes, they
are always mounted.

Next to the volumes, the storage module also tries to instantiate a cache. It
does this by creating a subvolume, preferably in a volume which was created
on SSD devices, and then creates a bind mount in the `/var` directory. The full
path of the cache is `/var/path`.
