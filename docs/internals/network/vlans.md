# VLANS

ZOS support vlans by allowing the farmer to setup vlan for both private and public subnets.

By default zos uses untagged traffic for both priv and public subnets (for both single or dual nic nodes). In some data centers and cloud providers, they can only provide tagged subnets.

ZOS can then become VLAN aware by providing optional vlan tags during booting.

## Private VLAN

Setting up private vlan forces zos to tag all private traffic with the configured vlan tag. This is possible by providing the `vlan:priv` kernel command line parameter

> Example `vlan:priv=302` will tag all private traffic with VLAN id `302`

During boot, zos tries to find the first interface that has ipv4 (over dhcp) normally all interfaces are probed until one of them actually get an IP. If a vlan ID is set, the probing also happen on the proper vlan, then the private default bridge (called `zos`) is then setup correctly with the proper vlan

```
                ┌────────────────────────────────────┐
                │              NODE                  │
                │                                    │
  vlan 302 ┌────┴──┐                                 │
───────────┤  Nic  ├──────────┐                      │
   tagged  └────┬──┘          │                      │
                │        ┌────┴─────┐                │
                │        │          │                │
                │        │   zos    │  pvid 302      │
                │        │   bridge ├──untagged      │
                │        │          │                │
                │        │          │                │
                │        └──────────┘                │
                │                                    │
                │                                    │
                │                                    │
                └────────────────────────────────────┘
```

## Public VLAN

> NOTE: Public VLAN in ZOS is **only** supported in a single nic setup. There is no support in dual nic yet

Setting up private vlan forces zos to tag all private traffic with the configured vlan tag. This is possible by providing the `vlan:pub` kernel command line parameter

> Example `vlan:pub=304` will tag all private traffic with VLAN id `304`

zos internally create a public bridge `br-pub` that can uses a detected ingress link (usually in dual nic setup) or shares
the same link as `zos` bridge by connecting to `br-pub` via a veth pair.

Single NIC setup

```
                ┌─────────────────────────────────────────────┐
                │                                             │
304 tagged ┌────┴─────┐                                       │
───────────┤   NIC    ├────────────┐                          │
           └────┬─────┘            │                          │
                │                  │                          │
                │          ┌───────┴─────┐                    │
                │          │             │                    │
                │          │      zos    │                    │
                │          │      bridge │                    │
                │          │             │                    │
                │          │             │                    │
                │          └───────┬─────┘                    │
                │                  │  pvid 304 untagged       │
                │                  │                          │
                │                  │                          │
                │           ┌──────▼─────┐                    │
                │           │            │                    │
                │           │    br-pub  │                    │
                │           │    bridge  │                    │
                │           │            │                    │
                │           │            │                    │
                │           │            │                    │
                │           └────────────┘                    │
                │                                             │
                └─────────────────────────────────────────────┘
```

## Dual NIC setup

Right now public vlans are not supported in case of dual nic setups. So in case public network is only available on the second nic then it will always be untagged traffic. This means the `vlan:pub` flag is silently ignored
