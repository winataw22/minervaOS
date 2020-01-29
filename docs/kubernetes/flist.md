# k3os flist

The [official zos k3os flist](https://hub.grid.tf/tf-official-apps/k3os.flist.md) contains the following binaries:

- k3os-vmlinux
- k3os-amd64.iso
- k3os-initrd-amd64

The `k3os-vmlinux` is a [custom built kernel](./kernel-config). `k3os-amd64.iso` and `k3os-initrd-amd64` are
binaries produced by running the `make` command in the [forked k3os repo](https://github.com/threefoldtech/k3os),
on the `zos-patch` branch.
