# `volume` type

`volume` is a virtiofs shared directory that can be mounted in a container or a virtual machine. Virtual machines must have virtiofs module enabled to be mounted properly. `volume` requires only `size` as specified [here](../../../pkg/gridtypes/zos/volume.go) and can only be used with `zmachine`.

It currently uses btrfs as the underlying file system to manage quota and supports extending its size without having to stop the `zmachine` attached to it.
