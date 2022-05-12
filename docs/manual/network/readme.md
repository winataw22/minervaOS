# `network` type
Private network can span multiple nodes at the same time. Which means workloads (`VMs`) that live (on different node) but part of the same virtual network can still reach each other over this `private` network.

If one (or more) nodes are `public access nodes` you can also add your personal laptop to the nodes and be able to reach your `VMs` over `wireguard` network.

In the simplest form a network workload consists of:
- network range
- sub-range available on this node
- private key
- list of peers
  - each peer has public key
  - sub-range

Full network definition can be found [here](../../../pkg/gridtypes/zos/network.go)

For more details on how the network work please refer to the [internal manual](../../internals/network/readme.md)
