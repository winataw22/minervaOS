# Zmachine

A `Zmachine` is an instance of virtual compute capacity. There are 2 kinds of Zmachines.
One is a `VM`, standard in cloud environments. Next to this it can also be a `container`.
On the Zos level, both of these are implemented as virtual machines. Depending on
the context, it will be considered to be either a VM or a container. In either
scenario, the `Zmachine` is started from an `Flist`.

## VM

A VM is a full blown virtualized environment capable of hosting an entire operating
system. The user has full control over the environment inside the VM, can change
files as they please, and can generally operate it as they would operate any server.

A VM reservations must also have at least 1 volume, as the boot image
will be copied to this volume. The size of the root disk will be the size of this
volume.

In the VM you are responsible for setting up the network and any kind of things
that need to be injected, such as SSH keys. For the networking, parameters will
be set on the kernel cmd line. You [can see a script that sets this up here](https://github.com/threefoldtech/cloud-container/blob/main/setupnetwork).

Environment variables will be injected on the kernel cmd line. It is your responsibility
to set up the image to read and parse the kernel cmd line, and take the proper action.

### Expected Flist structure

An `Zmachine` will be considered a `VM` if it contains a `/kernel` file. If this
is the case, it **MUST** also contain an `/image.raw` file. Having a `/kernel`
file without `/image.raw` is an error. Optionally, an `initramfs` image can be
provided as `/initrd` in the `Flist`.

`/kernel` is expected to be a `64-bit Linux` kernel (uncompressed). It can also
be a firmware blob implementing the `PVH` boot protocol. The hypervisor used in
Zos also supports `Windows 10/Windows server 2019`.

`/image.raw` is used as "boot disk". It should be noted that this is not currently
a traditional disk, but rather it is expected to be a `btrfs` filesystem. This has
some implications (see below). This `/image.raw` is copied to the first attached
volume of the `VM`. It is then loopback mounted on the host, so the filesystem can
be resized to the full size of the volume. Inside the `VM`, it is exposed as a disk,
and the kernel should mount it on `/`.

### Known issues

- The filesystem of the disk image must be `btrfs`. This excludes any kind of windows
    system.
- The disk image needs to be a filesystem, and can't be a full disk image.
- The kernel needs to be specified separately. It is not read from the disk image.
    As a result, you can't upgrade the kernel from inside the VM.
- The previous issue could be worked around by using a bootloader, but that doesn't
    work as those expect the disk image to have a partition table and EFI parition
    (which is usually some kind of `VFAT`). Recall that the disk image needs to be
    a single btrfs filesystem.
- Setting network is convoluted and very much not using __any__ industry standard.
- The kernel command line is just abused to pass configuration.

## Container

A container is meant to host `microservice`. The `microservice` architecture generally
dictates that each service should be run in it's own container (therefore providing
a level of isolation), and communicate with other containers it depends on over the
network.

In Zos, a container is actually also run in a virtualized environment. This is in
contract of the usual approach of running containers on the host. Similar to containers,
some setup is done on behalf of the user. After this is done, the users `entrypoint`
is started. It should be noted that a container has no control over the kernel
used to run it, if this is required, a `VM` should be used instead. Furthermore,
a container should ideally only have 1 process running. A container can be a single
binary, or a complete filesystem. In general, the first should be preferred, and
if you need the latter, it might be an indication that you actually want a `VM`.

For containers, the network setup will be created for you. Your init process can
assume that it will be fully set up (according to the config you provided) by the
time it is started. Mountpoints will also be setup for you. The environment variables
passed will be available inside the container.

### Expected Flist structure

Any Flist will boot as a container, **UNLESS** is has a `/kernel` file. There is
no need to specify a kernel yourself (it will be provided).

### Known issues

- The network config is injected over the kernel command line which clutters it,
    while there seems to be no reason that it is not passed via the environment
    variables.
- We need to do proper performance testing for `virtio-fs`. There seems to be some
    suboptimal performance right now.
- It's not currently possible to get container logs.
- TODO: more testing
