## Farmers providing transit for Tenant Networks (TN or Network)

For networks of a user to be reachable, these networks need penultimate Network resources that act as exit nodes for the WireGuard mesh.

For that Users need to sollicit a routable network with farmers that provide such a service. 

### Global registry for network resources. (`GRNR`?)

Threefold? shoud keep a store where Farmers can register also a network service for Tenant Network (TN) reachablility. 

In a network transaction the first thing asked should be where a user wants to purchase it's transit. That can be with a nearby (latency or geolocation) Farmer, or with a Farmer outside of the geolocation for easier routing towards the primary entrypoint. (VPN-like services coming to mind)

With this, we could envision in a later stage to have the network resources to be IPv6 multihomed with policy-based routing. That adds the possibiltiy to have multiple exit nodes for the same Network, with different IPv6 routes to them, 

### Datastructure

A registered Farmer can also register his (dc-located?) network to be sold as transit space. For that he registers:
  - the IPv4 addresses that can be allocated to exit nodes.
  - the IPv6 prefix he obtained to be used in the Grid 
  - the nodes that will serve as exit nodes.
  These nodes need to have IPv[46] access to routable address space through
    - Physical access in an interface of the node
    - Access on a public `vlan` or via `vxlan / mpls / gre`

Together with the registered nodes that will be part of that Public segment, the TNoDB can create a Network Object containing an ExitPoint for a Network.

Physcally Nodes can be connected in several ways:
  - living directly on the Internet (with a routable IPv4 and/or IPv6 Address) without Provider-enforced firewalling (outgoing traffic only)
    - having an IPv4 allocation --and-- and IPv6 allocation
    - having a single IPv4 address --and-- a single IPv6 allocation
  - living in a Farm that has Nodes only reachable through NAT for IPv4 and no IPv6
  - living in a Farm that has NAT IPv4 and routable IPv6 with an allocation
  - living in a single-segment having IPv4 RFC1918 and only one IPv6 /64 prefix (home Nodes mostly)

#### A Network resource allocation.
We define Network Resource as a routable IPv6 `/64` Prefix, so for every time a TNoDB needs to update the Tenant Network, to add a Network Resource to it, we need to get an allocation from the Farm Allocation.

Basically it's just a list of allocations in that prefix, that are in use. Any free Prefix will do, as we do routing in the exit nodes with a `/64` granularity. 

The TNoDB then creates/updates the Tenant Network object with that new Network Resource.

#### The Nodes responsible for ExitPoints 

A Node responsible for ExitPoints as wel as a Public endpoint will know so because of how it's registered in the Global Registry. That is :
  - it is defined as an exit node
  - the registry hands out an Object that describes it's public connectivity. i.e. :
    - the public IPv4 address(es) it can use
    - the IPv6 Prefix in the network segment that contains the penultimate default route
    - an eventual Private BGP AS number for announcing the `/64` Prefixes of a Tenant Network, and the BGP peer(s).

With that information, a Node can then build the Network Namespace from which it builds the Wireguard Interfaces prior to sending them in the ExitPoint Namespace.

So the Global Registry hands out
  - Tenant Network Objects 
  - Public Interface Objects

They are related :
  - A Node can have Network Resources
  - A Network Resource can have (1) Public Interface, and that ONLY when that NR is an ExitPoint
  - Both are part of a Tenant Network

A TNo defines a Network where ONLY the ExitPoint is flagged as being one. No more.  
When the Node (networkd) needs to setup a Public node, it will need to act differently.
  - Verify if the Node is **really** public, if so use standard WG interface setup
  - If not, verify if there is already a Public Exit Namespace defined, create WG interface there.
    - If there is Public Exit Namespace, request one, and set it up first.



