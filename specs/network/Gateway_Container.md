# Routing in ExitNodes

## Upstream Routers, ExitNodes, Gateway, IPv4 Nat

To get packets routed, you need to tell what goes where on every point that shuffles packets back and forth between interfaces.  
For everything that we control ourselves, things are simple, we know what goes where.  
It's more difficult to know for the upstream routers, where the Network Resources are located, i.e. behind what Exitpoint in what ExitNode.  
Keeping in mind that a `/48` allocation is 65536 IPv6 Prefixes (Subnets), it will be difficult for an upstream router to know all that, the more we receive a pseudo-random number for a prefix. Also, a lot of upstream routers (ASIC based) are totally incapable of handling 2^16 static routes. Another caveat is that although upstream routers can do bgp/ospf, they're not designed for such big routing tables, unless they're big Internet Backbone routers.
On the upstream part, we need it to be simple, no doubt about that, as having to run and maintain a routing protocol is going to be an unsurmountable problem for your Mom & Pop Farmer.

How we're using IPv6 (and IPv4) is not like 'normal' IPv6 planning goes in terms of slicing and dicing per department/locality/service...
Routes are per `/64` prefix and we need to be able to have them all. Also, 2^16 routes are.. erm.. not common, to say the least.

So how do we handle that?

First, on the upstream router, we need to make sure that as little as possible needs to be done and that it is already setup for scaling to bigger environments. Hence: as few routes as possible, and as simple to setup as possible.

Secondly, for now routing protocols __internally__ in the grid will generate more problems than they fix, because the dynamics in a grid will definitely put a strain on these routing protocols.

The requirements for a 'Farm' that provides for Tenant Network Exitpoints is that the 'Farmer' has an IPv6 allocation from their ISP and/or RIR for a `/48` network.
That gives us `65536` subnets to hand out to Network Resources. But that is a BIG number for static routes, so we need to split that in more manageable chuncks, luckily CIDR is made for that.

__(read this, it's important)__
A 'decent' linux system can handle a few 1000 routes, but we haven't seen what the upper limit would be. So let's split that `/48` in 16 equal parts, where the scaling can then be handeled by adding ExitNodes to handle routes.
So these 16 `/52` networks allow for 15 ExitNodes, the first part of the 16 being reserved for internal purposes.

Prefixes per ExitNode

```text
ExitNode1 -> 1000
ExitNode2 -> 2000
ExitNode3 -> 3000
ExitNode4 -> 4000
...
ExitNode10 -> a000
...
ExitNode14 -> e000
ExitNode15 -> f000
```

Every ExitNode's Gateway Container handles the routing for the ExitPoints in the ExitNode.
The gateway container is the routing endpoint (from the Upstream's point) for that 8th part. That way, in the Upstream router, there are only max 16 static routes.
Basically that allows for 4096 routes in the Gateway, that might aver to be very big in the long run, but for now we won't have to worry about that too much... Or maybe we do, we'll advise after the Alpha release.

### Network requirements and setup

There are requirements, of course.

- You have an IPv6 allocation and it's routed and announced.
- You have a small IPv4 allocation that is routable (not a single IP)
- ExitNodes need to have some decent network cards (Intel (I have Intel stock ;-) ))
- Your upstream Router can do IPv6 (really)
  - with PrefixZero configured and routable
  - with some of the 16 static routes added
- Your ExitNodes have at least 2 network cards.

#### Setting up Routes towards the ExitNode's Gateway Containers

Every router/l3 switch/firewall will have their specific ways to configure routing, but basically you'll need to add a route for every ExitNode you enabled, where the routes will always be deterministic, with only the allocation prefix as parameter. That makes things as simple as possible.

#### Gateway Containers

For the gateway containers to function properly, we can better have separate NICs in order to separate the effective bandwith for exit node execution (like 0-DB, and other Public-exitpoint IPv6 traffic) from Forwarding itself.
As routing in linux is not very cpu-hungry (provided you have decent NICs), we can easily run our router in the nodes themselves.

The other big advantage is that we can easily recreate the Gateway container and attached Exitpoints in another Node if one of the Exit nodes fails.

In a later phase it would be even (easily) possible to have the Gateway Containers in an active-active setup with VRRP or with some BGP or other routing protocol so that failure of an ExitNode doesn't bring down all the network resources attached to it, as there can be many (max 4k, but that is a lot).

#### PrefixZero

PrefixZero is the 0th network of an allocation but for us it's also extended to 1/16th of that allocation, so for the Prefixes of the NRs, we start at `$prefix:1000::/52` to `$prefix:f000::/52`, where every Exitnode handles the Exitpoints in that 'subnet'.


#### Implementation details

NOTE: on the right, you'll find the function names for handling the proper naming conventions

```ascii
         +-+ to [::]/0
         |      0.0.0.0/0
         |
+--------+---------------------------------------------------------------+
|                                                                        |
|                      UPSTREAM ROUTER                                   |
|    has                 routes                                          |
|    $prefixzero::1/64   $prefix:1::/52 via fe80::1000:0:0:1 dev swp1    |
|           fe80::1/64     or       via $prefix:0:1000:0:0:1             |
+------+-----------------------------------------------------------------+
       |
       |                                           PrefixZero
+------+---+--------------------------------------------------------+
           |
           |
           |                                   ExitNode1
+------------------------------------------------------------+
|          |         GatewayContainer                        |
| +--------------------------------------------------------+ |
| |        +                                               | |
| |   iface pub-1-1                                        | | GWPubName   (Gateway Routing iface)
| |        fe80::1000:0:0:1/64                             | | GWPubLL     (Gateway Route LL )
| |    $prefix:0:1000:0:0:1/64    (in prefixzero)          | | GWPubIP6    (Gateway Route IPv6)
| |    185.69.166.123/24                                   | | GWPubIP4    (Gateway IPv4 for -> SNAT only)
| |                                                        | |
| | iface to-1abc-1               iface to-1123-1          | | GWtoEPName  (ExitPoint's veth peer name)
| |    fe80::1:1abc/64               fe80::1:1123/64       | | GWtoEPLL    (Gateway to ExitPoint Link-Local)
| |    10.1.0.1/32                   10.1.0.1/32           | |
| |         +                            +                 | |
| +--------------------------------------------------------+ |
|           |                            |                   |
| +-----------------------+    +-----------------------+     |
| |         +             |    |         +             |     |
| |  iface  pub-1abc-1    |    |  iface  pub-1123-1    |     | EPPubName   (ExitPoint Pub iface name)
| |   fe80::1abc:1/64     |    |   fe80::1123:1/64     |     | EPPubLL     (ExitPoint Pub Link-Local)
| |  10.255.26.188/32     |    |  10.255.17.35/32      |     | EPPubIP4R   (ExitPoint Pub routing IPv4 on lo)
| | Network Resource      |    | Network Resource      |     |
| | - iface wg-1abc-1     |    | - iface wg-1123-1     |     | WGName      (Wireguard iface name)
| |   fe80::1abc/64       |    |   fe80::1123/64       |     | WGLL        (Wireguard Link-Local addr)
| |   10.255.26.188/24    |    |   10.255.17.35/24     |     | WGIP4RT     (Wireguard IPv4 routing address)
| | - iface veth-1abc-1   |    | - iface veth-1123-1   |     | NRLocalName (NR Local interface name )
| |   pref:1abc::1/64     |    |   pref:1123::1/64     |     | NRLocalIP6  (NR local IPv6 addr)
| |   10.26.188.1/24      |    |   10.17.35.1/24       |     | NRLocalIP4  (NR local IPv4 addr)
| |                       |    |                       |     |
| +-----------------------+    +-----------------------+     |
|   ExitPoint                     ExitPoint                  |
+------------------------------------------------------------+
                                               ExitNode1
 routes in Gateway (BAR)
 default via fe80::1 dev pub-1-1 # GWPubName(ExitnodeNR,AllocationNr)
 $prefix:1abc::/64 via fe80::1abc:1 dev to-1abc-1
 $prefix:1123::/64 via fe80::1123:1 dev to-1123-1

 routes in Exitpoint (only penultimate, the rest is handeled by the NR code)
 default via fe80::1:1abc dev pub-1abc-1
```


I hope the drawing speaks for itself, but in layman's terms, each ExitNode needs to become a BAR (big ass router). As BAR, it will be responsible for a 16th part of the Allocation a farmer has received.

We're speaking __always__ about a `/48`... Period. For other Allocations we'll explain afterwards in more elaborated documentation.

That basically means :

An Allocation prefix has 6 bytes:
`2a02:1802:005e:Exxx:yyyy:yyyy:yyyy:yyyy` where the numbers are fixed and routable to our router, and the `x & y` is what we get to play with.

Thus the 1st 4 `Exxx` is the number of subnets in `/64` we can route. A `/64` 'subnet' is a big (eufemism) address space, we admit, but it is the smallest unity in terms of physical segments in the IPv6 world.
So we get 2 bytes to address `/64` networks... That means 65536 Network Resources that we can route with that Allocation.
In router terms, that are **a lot** of routes, so we need to split that up. Even as BAR, 4K routes are many, but in the realms of feasible, so let's go for that.
One caveat, though, is that every time you get +2K routes, it is advisable to add an ExitNode in that Allocation. That is not 'bad' per se, but you will have to make sure that Node is properly set-up physically that it can act as one. In a standard Farm setup, we would even advise that every node in that farm is setup alike, so that any node can partake in these routing schemes, and thus that every node can act as backup for eventual failure of nodes that act as ExitNode.

So for any Network Resource in a Tenant Network the default is max 2 routing hops away, while the fanout is kept to a limited range.

Actions :

- When a __new__ Tenant network is created, __and__ the Node is an ExitNode, that is the place where the ExitPoint is created.
That means :
  - A Public namespace is created for instantiating the Wireguard interfaces that get sent into the ExitPoint
  - A Gateway Namespace (GW) is created (if it doesn't already exist) and the first route to that TN is added (the prefix of the Network Resource of that ExitPoint)
- The ExitPoint has a veth pair with one end into the Exitpoint and the other in the GW.
- Every time a TN is updated having a new NR, the GW updates it's routing table so that the new prefix points to the ExitPoint (which handles the routing for the TNs)
- In a later stage we'll define the GW as an NS with route-leaking VRFs per TN, so that we can forego having to maintain a routing/forwarding filter/firewall for the TNs.

TODO: figure out how we can update routing tables when NRs get removed from a TN as easily as possible (provisioning is easy, things get a lot more hairy when removing items)

TODO: we need to make this one level higher in terms of hierarchy: a (non-redundant for now) ExitNode can only get allocations from the TNoDB (whatever it's incantation) for which it is the GW. That means that when handing out a prefix for a new Network Resource, the TNoDB needs to take in account in what 16th it needs to allocate from.

On Node level, we know what to do:

- Updating a TNo for an Exitpoint, also updates the routing table in the BAR of the ExitNode.
- Every ExitNode is responsible for one 16th of the `/48` address space.
- The BAR does the IPv4 firewalling and NATing for the TN.
  
TODO:

- in terms of scalability for an ExitPoint, 3 byte addressing (4096) still seems a lot, but I don't think it would be a big issue, as arriving there would mean that a farm has 65536 possible network resources running, each with the possibility of 256 (IPv4) services in every Node where that TNo lives. That would be Amazon-big. But nibbling at bits on the TNoDB side would be sufficient to still adhere to our deterministic naming/allocations, so we can easily fix that.
- Wireguard being our first overlay, there are other overlays that can be more efficient, like vxlan.
- Sometimes overlay networks can be overkill and costly when we need to optimize for bandwidth, where dedicated NICs and vlans can make things static and still performant.

#### Opinionated, simplistic, deterministic, your pick

While we were reflecting on how we would have to generate IP addresses and destination ports of Wireguard interfaces, it seemed very complicated to have to maintain a relational database where living things needed to be registered and most of all, maintained. 

The network setup we envisioned needed to be 

- User driven
- Always adaptable as Network Resources (the IPv6 `/64` or IPv4 `/24` of a User (TNo) in a Node)  come and go.
- Easy to reason about
- Most of all, easy to debug in case something goes wrong

There so many combinations and incantations possible (this is the the case now, but will be even more so in the future) that having to maintain a living object with many relationships in terms of adding and/or deleting is not really mpossible, but very (extremely?) difficult and prone to errors. 
These errors can be User Errors, which can be fixable, but the most important problem is the possibility of discrepance between what is effectively live in a system and what is modeled in the database.
A part from that problem, to add insult to injury, upgrading a network with new features or different approaches, would add an increased complexity in migration of networks from one version(or form) to another. That as well in the model, as trying to reimplement the model to reality.
The more, a DataBase as single source of thruth adds the necessity to secure that database (with replacations, High Availability and all problems that are associated with maintianing databases). Needless to say, that is a problem that needs to be avoided like it were the plague. 

So:
We opted for trying to have a network as strict as possible. 

That is: a User has a Network with **only** a list of Node <-> Network Resource relationships. We need to be able to derive the **whole** network from solely these lists. No more, no less.

**`Uint16`**
The Magical Word in our network thingie. I assure you, we have come to **love** that word. It fits to a T for everything we envisioned, and never has let us down. (and we hope it never will)

Another word :

**Deterministic**
We live by it. We enforce it. And if we can't enforce, we find a way to (`sudo enforce`).

That means: from given data, make sure you can always (mostly unidirectionally) make sure that the **same** output can be used to apply a model. At he same time we tried to make sure it's still readable, or the we can apply the same programming rules in our mind and generate the outcomes ourselves, without the need of tools to do so.

The main advantage, though, having things set-up deterministically, is that we don't have to query a database to be able to apply a model. We only need a list. 
For our network, that list is of the form :

- a network has one ExitPoint AND that ExitPoint lives in a Node that is an ExitNode.
- an ExitPoint is the router for all Network Resources in that network (it is itself, by definition, also a Network Resource).
- the list of the Network Resources in that network is kept in BCDB.

That's it. The rest is derived. Interface names are derived. IP (as well IPv4 as IPv6) addresses are derived, Wireguard interfaces are derived. Routes are derived. Listening Ports are derived. Firewall rules are derived... you get the picture.
While that doesn't give a user a lot of leeway to shape things to his own ideas, it has the major benefit that the network can get out of a User's way. He doesn't have to think about it, and the network 'just' exists, just like the Internet does.  
Services started will be reachable for IPv6, and when a user owns IPv4 addresses, he can just PortForward into his network, as that network supports IPv4 too.

So: deterministic is our word for deriving parameters from just the Prefix Allocation that is assiggned to the Network Resource. The main advantage is that given the allocation received, everything falls automatically in place, and no queries to some database or shared state need to be done. 

It can be that purists don't like it, and there surely will be some even better ideas to get determinism going, but right now, it really works, and also gives us real ease to reason about how that thing is working. Implementing a complex structure that is a mesh of connections with their associated routes, keys, addresses, peers, ports gets a lot more easy to implement and verify correctness. That will be the same for tools to list and show the network structures in the (nearby) future.


