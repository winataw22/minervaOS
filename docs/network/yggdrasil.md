# Yggdrasil integration in 0-OS

Since day one, 0-OS v2 networking has been design around IPv6. The goal was avoid having to deal with exhausted IPV4 address and be ready for the future.

While this decision made sense on the long term, it pose trouble on the short term for farmer that only have access to ipv4  and are unable to ask for an upgrade to their IPS.

To workaround this problem, the idea is to have enough node in the grid with dual stack and leverage some of them to create some tunnels encapsulating ipv6 traffic. The ultimate goal being to ipv6 enable the nodes that needs it.

## Yggdrasil

[Yggdrasil network project](https://yggdrasil-network.github.io/) has been selected to be used to manages these tunnels.

## Integration in 0-OS

### networkd bootstrap

When booting, networkd will wait for 2 minute to receive an IPv6 address through router advertisement for it's `npub6` interface in the ndmz network namspace.
If after 2 minutes, no IPv6 is received, networkd will consider the node to be an IPv4 only nodes, switch to this mode and continue booting.

### 0-DB containers

For ipv4 only nodes, the 0-DB container will be exposed on top an yggdrasil IPv6 address. Since all the 0-OS node will also run yggdrasil, these 0-DB container will always be reachable from any container in the grid.

For dual stack nodes, the 0-DB container will also get an yggdrasil IP in addition to the already present public IPv6.

### Containers

In a dual stack node, all container are dual stack too.

The situation is not the same for ipv4 only nodes. To enable those nodes to also access the IPV6 network, yggdrasil will be used to create tunnel transporting ipv6 traffic to an "exit node". The traffic will exit and send to the internet from the exit node and the traffic coming back will the transported back to the origin node through the tunnel.

Farmer will then be able to configure some of their node to become tunnels providers. This means the farmer will allocate a /64 ipv6 prefix that can be used by ipv4 only nodes to create those tunnels.