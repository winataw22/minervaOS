# Yggdrasil integration in 0-OS

Since day one, 0-OS v2 networking has been design around IPv6. The goal was avoid having to deal with exhausted IPV4 address and be ready for the future.

While this decision made sense on the long term, it pose trouble on the short term for farmer that only have access to ipv4  and are unable to ask for an upgrade to their IPS.

In order to allow these ipv4 only nodes to join the grid, an other overlay network has to be created between all the nodes. To achieve this, Yggdrasil has been selected.

## Yggdrasil

[Yggdrasil network project](https://yggdrasil-network.github.io/) has been selected to be integrated into 0-OS. All 0-OS node will runs an yggdrasil daemon which means all 0-OS nodes can now communicate over the yggdrasil network. The yggdrasil integration is an experiment planned in multiple phase:

Phase 1: Allow 0-DB container to be exposed over yggdrasil network. Implemented in v0.3.5
Phase 2: Allow containers to request an interface with an yggdrasil IP address.

## networkd bootstrap

When booting, networkd will wait for 2 minute to receive an IPv6 address through router advertisement for it's `npub6` interface in the ndmz network namspace.
If after 2 minutes, no IPv6 is received, networkd will consider the node to be an IPv4 only nodes, switch to this mode and continue booting.

### 0-DB containers

For ipv4 only nodes, the 0-DB container will be exposed on top an yggdrasil IPv6 address. Since all the 0-OS node will also run yggdrasil, these 0-DB container will always be reachable from any container in the grid.

For dual stack nodes, the 0-DB container will also get an yggdrasil IP in addition to the already present public IPv6.