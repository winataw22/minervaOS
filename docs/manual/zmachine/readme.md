# `zmachine` type

`zmachine` is a unified container/virtual machine type. This can be used to start a virtual machine on a `zos` node give the following:
- `flist`, this what provide the base `vm` image or container image.
  - the `flist` content is what changes the `zmachine` mode. An `flist` built from a docker image or has files, or executable binaries will run in a container mode. `ZOS` will inject it's own `kernel+initramfs` to run the workload and kick start the defined `flist` `entrypoint`
- private network to join (with assigned IP)
- optional public `ipv4` or `ipv6`
- optional disks. But at least one disk is required in case running `zmachine` in `vm` mode, which is used to hold the `vm` root image.

For more details on all parameters needed to run a `zmachine` please refer to [`zmachine` data](../../../pkg/gridtypes/zos/zmachine.go)

# Building your `flist`.
Please refer to [this document](zmachine.md) here about how to build an compatible `zmachine flist`
