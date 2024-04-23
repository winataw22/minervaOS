# Zmachine

A `Zmachine` is an instance of virtual compute capacity. There are 2 kinds of Zmachines.
One is a `VM`, standard in cloud environments. Next to this it can also be a `container`.
On the Zos level, both of these are implemented as virtual machines. Depending on
the context, it will be considered to be either a VM or a container. In either
scenario, the `Zmachine` is started from an `Flist`.

> Note, both VM and Container on ZOS are actually served as Virtual Machines. The
only difference is that if you are running in VM mode, you only need to provide
a kernel at `/boot/vmlinuz` in your flist.

## Container

A container is meant to host `microservice`. The `microservice` architecture generally
dictates that each service should be run in it's own container (therefore providing
a level of isolation), and communicate with other containers it depends on over the
network.

Similar to docker. In Zos, a container is actually also run in a virtualized environment.
Similar to containers, some setup is done on behalf of the user. After setup this is done,
the users `entrypoint` is started.

It should be noted that a container has no control over the kernel
used to run it, if this is required, a `VM` should be used instead. Furthermore,
a container should ideally only have 1 process running. A container can be a single
binary, or a complete filesystem. In general, the first should be preferred, and
if you need the latter, it might be an indication that you actually want a `VM`.

For containers, the network setup will be created for you. Your init process can
assume that it will be fully set up (according to the config you provided) by the
time it is started. Mountpoints will also be setup for you. The environment variables
passed will be available inside the container.

## VM

In container mode, zos provide a minimal kernel that is used to run a light weight VM
and then run your app from your flist. If you need control over the kernel you can actually
still provide it inside the flist as follows:

- /boot/vmlinuz
- /boot/initrd.img [optional]

Any of those files can be a symlink to another file in the flist.

If ZOS found the `/boot/vmlinuz` file, it will use this with the initrd.img if also exists. otherwise zos will use the built-in minimal kernel and run in `container` mode.

### Building an ubuntu VM flist

This is a guide to help you build a working VM flist.

This guide is for ubuntu `jammy`

prepare rootfs

```bash
mkdir ubuntu:jammy
```

bootstrap ubuntu

```bash
sudo debootstrap jammy ubuntu:jammy  http://archive.ubuntu.com/ubuntu
```

this will create and download the basic rootfs for ubuntu jammy in the directory `ubuntu:jammy`.
After its done we can then chroot into this directory to continue installing the necessary packages needed and configure
few things.

> I am using script called `arch-chroot` which is available by default on arch but you can also install on ubuntu to continue
the following steps

```bash
sudo arch-chroot ubuntu:jammy
```

> This script (similar to the `chroot` command) switch root to that given directory but also takes care of mounting /dev /sys, etc.. for you
and clean it up on exit.

Next just remove this link and re-create the file with a valid name to be able to continue

```bash
# make sure to set the path correctly
export PATH=/usr/local/sbin/:/usr/local/bin/:/usr/sbin/:/usr/bin/:/sbin:/bin

rm /etc/resolv.conf
echo 'nameserver 1.1.1.1' > /etc/resolv.conf
```

Install cloud-init

```bash
apt-get update
apt-get install cloud-init openssh-server curl
# to make sure we have clean setup
cloud-init clean
```

Also really important that we install a kernel

```bash
apt-get install linux-modules-extra-5.15.0-25-generic
```

> I choose this package because it will also install extra modules for us and a generic kernel

Next make sure that virtiofs is part of the initramfs image

```bash
echo 'fs-virtiofs' >> /etc/initramfs-tools/modules
update-initramfs -c -k all
```

clean up cache

```bash
apt-get clean
```

Last thing we do inside the container before we actually upload the flist
is to make sure the kernel is in the correct format

This step does not require that we stay in the chroot so hit `ctr+d` or type `exit`

To verify you can do this:

```bash
ls -l ubuntu:jammy/boot
```

and it should show something like

```bash
total 101476
-rw-r--r-- 1 root root   260489 Mar 30  2022 config-5.15.0-25-generic
drwxr-xr-x 1 root root       54 Jun 28 15:35 grub
lrwxrwxrwx 1 root root       28 Jun 28 15:35 initrd.img -> initrd.img-5.15.0-25-generic
-rw-r--r-- 1 root root 41392462 Jun 28 15:39 initrd.img-5.15.0-25-generic
lrwxrwxrwx 1 root root       28 Jun 28 15:35 initrd.img.old -> initrd.img-5.15.0-25-generic
-rw------- 1 root root  6246119 Mar 30  2022 System.map-5.15.0-25-generic
lrwxrwxrwx 1 root root       25 Jun 28 15:35 vmlinuz -> vmlinuz-5.15.0-25-generic
-rw-r--r-- 1 root root 55988436 Jun 28 15:50 vmlinuz-5.15.0-25-generic
lrwxrwxrwx 1 root root       25 Jun 28 15:35 vmlinuz.old -> vmlinuz-5.15.0-25-generic
```

Now package the tar for upload

```bash
sudo rm -rf ubuntu:jammy/dev/*
sudo tar -czf ubuntu-jammy.tar.gz -C ubuntu:jammy .
```

Upload to the hub, and use it to create a Zmachine

## cloud-console

`cloud-console` is a tool used to interact with Zmachines deployed through 0-OS. It manages to connect to VMs over `pseudoterminal` (`pty`) exposed by `cloud-hypervisor`.
For more details on how it is integrated with 0-OS check out [cloud-console](./cloud-console.md).
For more details on `cloud-console` itself and how it works check out [cloud-console](https://github.com/threefoldtech/cloud-console).
