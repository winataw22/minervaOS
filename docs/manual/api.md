# API
This document should list all the actions available on the node public API. which is available over [RMB](https://github.com/threefoldtech/rmb-rs)

The node is always reachable over the node twin id as per the node object on tfchain. Once node twin is known, a [client](../../client/node.go) can be initiated and used to talk to the node.

# Deployments
## Deploy
| command |body| return|
|---|---|---|
| `zos.deployment.deploy` | [Deployment](../../pkg/gridtypes/deployment.go)|-|

Deployment need to have valid signature, the contract must exist on chain with the correct contract hash as the deployment.

## Update
| command |body| return|
|---|---|---|
| `zos.deployment.update` | [Deployment](../../pkg/gridtypes/deployment.go)|-|

The update call, will update (modify) an already existing deployment with new definition. The deployment must already exist on the node, the contract must have the new hash as the provided deployment, plus valid versions.

> TODO: need more details over the deployment update calls how to handle the version

## Get
| command |body| return|
|---|---|---|
| `zos.deployment.get` | `{contract_id: <id>}`|[Deployment](../../pkg/gridtypes/deployment.go)|

## Changes
| command |body| return|
|---|---|---|
| `zos.deployment.changes` | `{contract_id: <id>}`| `[]Workloads` |

Where:
 - [workload](../../pkg/gridtypes/workload.go)

The list will contain all deployment workloads (changes) means a workload can (will) appear
multiple times in this list for each time a workload state will change.

This means a workload will first appear in `init` state, then next time it will show the state change (with time) to the next state which can be success or failure, and so on.
This will happen for each workload in the deployment.

## Delete
> You probably never need to call this command yourself, the node will delete the deployment once the contract is cancelled on the chain.

| command |body| return|
|---|---|---|
| `zos.deployment.get` | `{contract_id: <id>}`|-|

# Statistics
| command |body| return|
|---|---|---|
| `zos.statistics.get` | - |`{total: Capacity, used: Capacity, system: Capacity}`|

Where:

```json
Capacity {
    "cur": "uint64",
    "sru": "bytes",
    "hru": "bytes",
    "mru": "bytes",
    "ipv4u": "unit64",
}
```

> Note that, `used` capacity equal the full workload reserved capacity PLUS the system reserved capacity
so `used = user_used + system`, while `system` is only the amount of resourced reserved by `zos` itself

# Storage
## List separate pools with capacity
| command |body| return|
|---|---|---|
| `zos.storage.pools` | - |`[]Pool`|

List all node pools with their types, size and used space
where
```json
Pool {
    "name": "pool-id",
    "type": "(ssd|hdd)",
    "size": <size in bytes>,
    "used": <used in bytes>
}
```
# Network
## List Wireguard Ports
| command |body| return|
|---|---|---|
| `zos.network.list_wg_ports` | - |`[]uint16`|

List all `reserved` ports on the node that can't be used for network wireguard. A user then need to find a free port that is not in this list to use for his network

## Supports IPV6
| command |body| return|
|---|---|---|
| `zos.network.has_ipv6` | - |`bool`|

## Interfaces
| command |body| return|
|---|---|---|
| `zos.network.interfaces` | - |`map[string][]IP` |

list of node IPs this is a public information. Mainly to show the node yggdrasil IP and the `zos` interface.

## List Public IPs
| command |body| return|
|---|---|---|
| `zos.network.list_public_ips` | - |`[]IP` |

List all user deployed public IPs that are served by this node.

## Get Public Config
| command |body| return|
|---|---|---|
| `zos.network.public_config_get` | - |`PublicConfig` |

Where
```json
PublicConfig {
    "type": "string", // always vlan
    "ipv4": "CIDR",
    "ipv6": "CIDR",
    "gw4": "IP",
    "gw6": "IP",
    "domain": "string",
}
```

returns the node public config or error if not set. If a node has public config
it means it can act like an access node to user private networks

# Admin
The next set of commands are ONLY possible to be called by the `farmer` only.
## Interfaces
| command |body| return|
|---|---|---|
| `zos.network.admin.interfaces` | - |`map[string]Interface` |

Where
```json
Interface {
    "ips": ["ip"],
    "mac": "mac-address",
}
```

Lists ALL node physical interfaces.
Those interfaces then can be used as an input to `set_public_nic`

## Get Public Exit NIC
| command |body| return|
|---|---|---|
| `zos.network.admin.get_public_nic` | - |`ExitDevice` |

Where
```json
ExitInterface {
    "is_single": "bool",
    "is_dual": "bool",
    "dual_interface": "name",
}
```

returns the interface used by public traffic (for user workloads)
## Set Public Exit NIC
| command |body| return|
|---|---|---|
| `zos.network.admin.set_public_nic` | `name` |- |

name must be one of (free) names returned by `zos.network.admin.interfaces`

# System

## Version
| command |body| return|
|---|---|---|
| `zos.system.version` | - | `{zos: string, zinit: string}` |

## DMI
| command |body| return|
|---|---|---|
| `zos.system.dmi` | - | [DMI](../../pkg/capacity/dmi/dmi.go) |

## Hypervisor
| command |body| return|
|---|---|---|
| `zos.system.hypervisor` | - | `string` |
