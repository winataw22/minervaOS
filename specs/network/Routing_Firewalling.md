## Routing in Circular Meshes with Wireguard
____

As we are leveraging WireGuard as building block for communications that are inter-node **per** user network, each Network Resource receives it's own IPv6 Prefix, and eventually it's own RFC1918 private IPv4 subnet.

Either way we want IPv6 to function, there will be a need for that circular network to be able to ultimately have a route to the BBI (Big Bad Internet). So every User Network will have an NR that is also an exit, with the appropriate default gateway, being an upstream route.

**READ THIS, it's important**  
_That also means that the Upstream Router needs to know the route back to the Prefixes/subnets that live in that circular net._ (more on that later)

Also, [Get  yourself a little acquainted with IPv6]()

### Network Resource Containers
----

In the Network Resource Container, a single WireGuard interface is set-up with every network resource of that user network as peer, with `AllowedIPs` containing the Link-Local IPv6 address and Prefix of that Peer.
These prefixes are exactly reflected as routes in the Network Resource Container, in time we could als envision adding IPv4, where bits 8-23 are deterministically derived from the Prefix's last 2 nibbles (a nibble is one byte in an IPv6 address).

For example :

  - Considering we received an IPv6 Allocation of : `2a02:1802:5e::/48`
    - that gives us possibility for `64 - 48 = 16` -> `2^16 = 65536` possible network resource containers, each having a `/64` Prefix. In short, even for a decent built out farm, that should largely suffice... although... But then it's just making sure you request an even decent-er allocation ;-)

  - A valid Prefix for an NR is: `2a02:1802:5e:010a::/64`
    - having `2a02:1802:5e:10a::1/64`
    - and having : `10.1.10.254/24` (`01 = 1 and 0a = 10`)
    - for IPv4 routing to work, the WireGuard interface will also need an IPv4 address (`169.254.1.10/16`)

  - Every NR Container has a route for all Peer Prefixes/subnets.
  - Every NR Container has a default route pointing to the NR that is an exit node

### Exit Containers
----

The Exit Container is basically an NR with an added interface and Link-Local IPv6 address that is connected to the Upstream network segment, be it IPv6 or IPv4.  
For IPv6, in terms of routing, it has only a default gateway, so it can send packets on their merry way for all routes its has no knowledge of.  
TBD : will this be a separate container, connected to a bridge that itself is also connected to an NR... 
  - So the exit network interface (a veth pair, an OVS port, a physical interface (like an SRIOv VF)):
    - has a link-local address on the routing segment (IPv6/4)
    - has a default route configured in its NR Container pointing to the link-local add of the upstream router (AKA unnumbered routing, but for IPv4 you need a real IP addr)

### Upstream Routers / L3-Switches
----

In order for manually set-up routes to work properly, both ends of each peer need to know their routes.  
  - For an exit container, that's easy: a defalt route.
  - For the Upstream router (the machine/switch/router having the ultimate default route), each allocated prefix to an NR needs to be forwarded to the link-local addr of the exit container of the Network (that Circular Mesh) containing the NR for packets to be able to return.  
  That will become a big problem in terms of where we will host the main router for a set of prefixes or for an allocation.

  - #### Routing with Linux and routing table sizes
To address routing to exit nodes of a Network (i.e. a group of prefixes) we need to register the exit node to a machine/container(?) so that all traffic for a network gets forwarde to the link-local address of that exit container.  

Let's say to start we're using a container, on linux, where it has two NICs, one in the segment that has an allocated Prefix of `/48` that we can slice and dice tou our hearts content, and one that is used for forwarding out, into the ZOS Networks.
Ergo: we have received `2a02:1807:1100::/48` that we can slice up.

On the outside we face our Provider, that (mostly) has set up a filter in his router that we are allowed to announce our allocation. Do I need to specify here that we have to run BGP ? Nahh...

So somewhere in our env we will have an FRR running with a simple BGP config to announce our allocation. Now you might be pressed to say that we'll just request a static route and work from there, but then what to do with a provider who is adamant that we won't get any. Their way or the highway? I suspect we'll be on the highway a lot. Anyway ... static it is.

  - #### Static Routing
Imagine something like this: 

We received form our Provider a /48 and that Provider exceptionally set up a static route for that /48 to point to a real numbered interface.

On our end, either way we need a router. Let's assume we have a Linux box to handle that.  

```
     Static Routing


         +-----------------------------+
         |       Provider router       |
         |                             |
         |                             |
         |   2a02:1800:10:1052::4/127  |
         +-------------+---------------+
                       |
                       |
                       | Linux Routing Container
         +-------------+---------------+
         |   2a02:1800:10:1052::5/127  |
         |                             | Default route for ::/0 via 2a02:1800:10:1052::4
         |  prefix = 2a02:1802:5e (/48)| Depending on Provider, that can also be
         |  $prefix::1/64              | unnumbered on interface (using fe80:: with
         +-----------------------------+ multicast addrs)


$prefix::1:1/64  $prefix::a:1/64  $prefix::a1c2:1/64
  +-----------+   +-----------+   +-----------+
  | NR  Exit  |   |           |   |           |        (watch the placement of :: )
  |           |   |           |   |           |
  |           |   |           |   |           |
  |           |   |           |   |           |
  +-----------+   +-----------+   +-----------+
$prefix:1::1/64  $prefix:a::1/64  $prefix:a1c2::1/64

```

The main concern will be that we make that networking as idempotent as possible, to be able to recreate exactly the same config on another node in case of failure of the router node

If for an Exit container we have an allocation of `$prefix:aaaa::/64` :
  - the router facing IP will be : `$prefix::aaaa:1/64` with default gw `$prefix::1`
  - the wireguard interface has : `$prefix:aaaa::1/64`

NOTE: we're working here with an Allocation of `/48` which makes life easy with the 1st 6 nibbles as a static thing, but any other allocation works, where e.g a `/40` is more like:

`2a02:1802:0000:: to 2a02:1802:00ff::`

----
But we have already an allocation and an AS in RIPE (RIR for Europe,Russia and Middle-East)  
[Reference here](https://en.wikipedia.org/wiki/Regional_Internet_registry)

ThreeFold allocation ;-)  
prefix = `2a05:2380::/29`  

The allocation is `2a05:2380:: to 2a05:2387:ffff:ffff:ffff:ffff:ffff:ffff`  
In short +- 34 Billion /64 networks  (yes, Beelion, like, a lot).  
Since we have a `/29`, we can divide it up in 2048 `/40`, each having +16 million `/64`.

To put that a little bit in perspective : In Europe alone, we can have 400+ node farms, where each farm having 16 million Network Resources of size `/64`. That would be that each node can have +40000 Network Resources that have a `/64` for services to run. (O_o)

The more, we will be easily allowed by Providers to announce a `/40`, as the smallest announcement on IPv6 is a `/48`  
That is : for __our__ Ripe allocation, but most DC's will just give us a `/40` with the flick of a pen.

That as a side-note.  
[If you want to play with numbers a little, go here ;-)](http://www.gestioip.net/cgi-bin/subnet_calculator.cgi).  
[Or look at big numbers, they're mind boggling](https://www.mediawiki.org/wiki/Help:Range_blocks/IPv6)

----
To have a route back to our exit node we need to handle these routes: every exit node has it's own routes to the Network Resource `/64`'s behind the wireguards.  
So we can leverage the User Network Object that is stored somewhere to insert the routes to the prefixes of the User Network.

Seen that there are linux boxes that can contain a full BGP database, we can assume that adding 1000's of routes into a kernel will not be a big problem, but these tests have not been done, we can only surmise that it will work.

Also havind __all__ routes in a container is not really necessary, as we can configure multiple router containers to be responsible for only a part fo the User Networks.

This also gives us an easy way to do self healing, in case a Node containing the Routing Container fails, as we can just reapply the same User Network Objects to a new container in another node.

### IP Address Allocations for containers
-----
Most Containers (or services running in containers) will 'just' need to attach to the Network Resource container (the one holding the wireguard tunnel) and get on with it, but some might need a separate IP to be able to listen on a conflicting port (or any other constraint).  
Then that Container gets created with a veth pair and attached to the (already existing) bridge for that network resource.  
Rest now to know how to allocate an IP address to that container, and in case of a VM, how that vm can get an IP address (dnsmasq on the NR Container)? Or a DHCP Relay to the dnsmasq of an exit container? Are we missing things in the Network struct?

## Service discovery/registration of DNS.
----

This is a hot point, as with the arrival of IPv6, a user will be hard pressed to always have to copy-paste (or god forbid, TYPE in) that address...

We need to address that. We're missing:
  - how will a service (running flist with an UTS namespace) be registered and associated with a Network Resource?
  - where will these services be registered so they can be found by hostname or SRV record
  - how will the Network Resources do DNS, where are these nameservices going to run
  - ... etc ...


