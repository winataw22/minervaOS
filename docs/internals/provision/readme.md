# Provision Module

## ZBus
This module is autonomous module and is not reachable over `zbus`.

## Introduction

This module is responsible to provision/decommission workload on the node.

It accepts new deployment over `rmb` and tries to bring them to reality by running a series of provisioning workflows based on the workload `type`.

`provisiond` knows about all available daemons and it contacts them over `zbus` to ask for the needed services. The pull everything together and update the deployment with the workload state.

If node was restarted, `provisiond` tries to bring all active workloads back to original state.
## Supported workload

0-OS currently support 8 type of workloads:
- network
- `zmachine` (virtual machine)
- `zmount` (disk): usable only by a `zmachine`
- `public-ip` (v4 and/or v6): usable only by a `zmachine`
- [`zdb`](https://github.com/threefoldtech/0-DB) `namespace`
- [`qsfs`](https://github.com/threefoldtech/quantum-storage)
- `zlogs`
- `gateway`
