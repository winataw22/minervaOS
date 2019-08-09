## Preamble
-----------
A 0-OS node is defined as the highest granularity of a DC; i.e. every node
implements it's proper reachability for containers/vms that are running in it.

That means that 0-OS needs to be able to set up networking and firewalling for
separated tenant loads.

Also it will be necessary that every node can then handle it's own IPv6
allocations for the tenant networks.

A tenant load is defined as a service/application that gets exposed over IPv6,
so a tenant network gets a /64 allocation __for each node__ , connected in a
pause container and/or bridge.

For tenant networks to access each other over 0-OS (node) boundaries, a
Wireguard tunnel is built between the tenant bridges, and static routing is
configured.

For each tenant network, there is one exit point, where a tenant could buy an
IPv4 resource if it's available.

Also, we quote 'Farm' as in terms of 0-OS, there is no such thing, except maybe in a later phase where we can define some physical nodes (if they are living in a NATed IPv4 private space) to be part of a 'Farm' Network.  
But right now, we have as a requirement for Farmers that they need to reqest an IPv6 allocation from the Datacentre where they'll host their servers. Then it is already a given that nodes local to a farm will have shortest paths to the neighbor nodes.


## Networking Considerations for 0-OS deployment
-------------

1. ### Nodes deployed in a DC that provides for IPv4 allocations ONLY

Note: this is not appicable for now, we require nodes in a datacentre to have IPv6.

That would mean that tenant networks needs to build an 6in4 tunnel to some IPv6
exit point, AND thus have an allocation from the exit point's IPv6 prefix.

On the other hand the IPv4 exit can be handled locally, with a proper IPv4 proxy
into the IPv6 service. So only the exit container/router/gateway needs to be
dual-stack, ever.

2. ### Nodes deployed in a DC that is dual-stack (IPv4/6)

Tunnels 6in4 then do not apply, and IPv4 proxies can be handled locally.

IPv6 allocations are simple to come by, it's just a matter of requesting an
allocation from the LIR (Local Internet Registrar). Mostly they will give a /48 to start with, that gives ample space already. (65536 subnets).   
As the smallest advertised (BGP) prefix for an allocation is a /48, when explaining to the network admin from the DC what your IPv6 Requirements are, they'll gladly give you a /40. And if that doesn't suffice, you can just ask a /32 from the RIR, the IPv6 space is so vast that it's only an administrative entry. (a /32 gives you 2^32 /64 prefixes, that is as many as the full IPv4 Address space).  

We then need to be able to hand out an allocation from that range for a tenant
service/network and announce a route, e.g. upstream
`ip route add 2001:fe80:aaaa:1234::/64 via 2001::fe80:aaaa::1234`, which means:
subnet `xxxx/64` is reachable via node `yyyy` (note the placement of the '`::`').

But that kind of administration for aquiring IPv6 allocations need to be done and will require both good documentation of how to obtain them and how to register these resources on the grid to be usable.

**Routing**:  
In the node's main table that route points to the IP addr of the bridge of that
tenant network (here `2001:fe80:aaaa:1234::`, note: `::` is an ip address, as
there are no 'network numbers' in IPv6). In the implementation details that is called `PrefixZero`

Firewalling for a destination service will still be ip/port, and IPv4 exposure
can then only be through a proxy for that service.

(There are IPv4->IPv6 nat implementations, but they are flaky at best)

3. ### Nodes deployed in a HOME ipv4-only NAT network

Same as 1. , but where a local IPv4 proxy is not possible, and a 6in4 tunnel to a nearby IPv6 exit, that encompasses an IPv4 service proxy

4. ### Nodes in (more modern) home networks that also have an IPv6 /64 allocation.

Because we need more than one /64 prefix, and that the average home IPv6 router is an outgoing-only firewall, we'll need to to the same as 3. , eventually later do 6in6 for tunneling. All the rest of 3. applies.


# Simplicity

Ultimately, we want to make things as simple as possible, but (sic Albert) not one bit simpler.  
That means that, because of the very existence of the Internet, we can basically piggyback as much as possible on the existing infrastructure, where interconnectivity is already taken care of.

In order to keep things simple (for users, that is), the idea would be to have an IPv6-only network up until some 4->6 translation (be it over L3 or L4/7) for services that need to be reachable over IPv4. 

Of course, for IPv6 to be a viable thing, every service will need to be named, as it is virtually impossible to use IPv6 addresses as service endpoint definitions. IPv6 addresses need to be resolved by name services in any case. That also means that all applications need to be able to resolve proper AAAA records (DNS parlance for IPv6 records).

## Routing

When using IPv6, we'll need to handle routing to the services that have by definition their own IPv6 address, noting that a 'service' is mostly a container with a single process running in it.

Let's start with the basics: 

A 'service' encompasses an IPv6 address and a listening port. That basically means that that service is capable to listen on a specific ip address/port/interface. Most modern applications do nowadays, but YMMV.  
So if you have a binary that needs to run and that binary is not capable of setting it's listen address/port, you'll need to run that in a separate network namespace.

Routing needs to be established in 2 directions up until you hit a default route. That basically means that an endpoint (service) has effectively no routes except its default route and it's local network. Simple enough. (if I can't talk to someone in the room, I'll send my message out through the door).  

But the other way around tends to get less simple.

IPv6 allocations differ per site and availabilty. 
  - in a DC that hands out IPv6 allocations, you can get them as well static as through BGP. Most ISP's/DC's will require that you request an allocation from the RIR , or if a DC is a LIR they can allocate one to you from their pool. Mostly you'll get a /48, i.e. 16K subnets.  
  In IPv6-land static routing is severely frowned upon, and most will just setup a filter in their router so that they allow you to announce your full prefix to your BGP peer(the provider's router). Hence: in 90% of the cases, you'll be bound to have a BGP router installed and configured, from where you can subclass your prefix. 

  - in a home network with an ISP-provided wifi AP/router, there will be 'just' a (one) `/64` available, so if you would wish to provide the same type of networking setup to a node in a home network, we'll need to establish a tunnel to an exit with a prefix that is routable through that tunnel.  
  TODO: more explanation.

  - the same applies for home routers that only have IPv4 (e.g 192.168.0.0/24).


## IPv4 <-> IPv6 communications

1. ### Implement the Network Resources(NR) as dual-stack 
We can of course also have the NRs as dual-stack (carrying both IPv6 as IPv4), where the IPv4 part can then be penultimate routed to the Exitpoint, and NATed to the Exterior, whereas the IPv6 part is fully routed.  
The implemanetation already is dual-stack, where the IPv4 addresses are derived from the local IPv6 prefix of the allocation a node got for that specific Tenant's network.

2. ### Tunneled networks with ULA prefixes for private networks and NAT
No need to work for this, as that can be viewed in the future

3. ### Jool as a means to NAT ipv6 to ipv4 (snat/dnat) and vice-versa
For now we're going to cross these boundaries with proxies, the containers will default to IPv6 for going out.

## IPv6 prefixes for tenant networks (l2/l3) and prefix delegations.

However a tenant network or endpoint providing a service is set-up, we need to establish on how we will interconnect a user's network. 

Given that we want to define that a node is a completely stand-alone entity, we will also need to establish if and how we want to segregate multiple users their networks. It will be important that some networks are publicly reachable and some will only be reachable for specific networks.

There are several ways to do this, each with their proper caveats and difficulties.

But most of all things will need some configuration management that is stored __somewhere__. Every prefix has a link to a user __and__ a list of tunnel endpoints, with their routable prefixes.

![Preliminary layout](Nodes.svg)

1. ### Pure IPv6 routing with handling of firewall rules for each service/prefix/IP/port

A Tenant Network is a group of prefixes per user of the grid. Each prefix is owned by a node. In a node, services of the same user live within the same prefix, be it in a pause container, over a bridge , or in a vm connected to that bridge.  
For this to work properly there are a few requisites:
  - Who is going to hand out a prefix that is :
    1. part of the prefixes allocated to a node
    2. will be grouped into a network of prefixes for a user
  - Where do we store prefix allocations for a node/site:
    1. When in a DC and we received an allocation (like a /48 or even /40)
    2. When a node is deployed in a home network with or without IPv6. That means that we need to know where we'll have an exit container running, have the tunnel set-up, and have an allocation for the prefix.
    3. When multiple nodes are interconnected from home networks, we'll also need to provide for some hole punchers for the wireguards in the pause containers of the users.

2. ### Crossing public/private boundaries

Networking has basically two modi operandi.  
  - A public mode where the network packets themselves are not encrypted and can arrive to a server where the application can choose wether or not to encrypt data.
  - A private mode, where network packets and their data contents are protected from too curious eyes. Mostly the protected networks are physically separated from the public networks, where the boundaries are protected by firewalls.

To emulate that public/private concept in an environment where the physical connections can't be trusted, be it malevolent or by ignorance, we need to make sure that everything that is not public network gets encrypted on the wire, so that noone can snoop on the physical cable and steal data.  
Separating public from private can then be done properly by firewalls at the boundaries.  

So:
For public communications we don't have to do anything, except set-up the exit node (basically a containter) and make sure it's reachable. The packets already traversed an untrusted network, we can keep it that way.  
But for packets that are supposedly private but traverse an untrusted network, we'll need to encrypt the packets before they hit the wire, as in private networks applications are typically not encrypted.  
For every part of services in a node that is from a user, each part needs to connect to the same contex in terms of networking to all other nodes where the user has services that need to communicate. To do that, the grid sets up a circular mesh of wireguard tunnels, where one point on the circle is an exit/firewall.  

The IPv6 allocations we receive are like the one we received in the good'ol times where every PC had a public IP address. They are globally routable. There is no NAT (TODO explain NAT), none, zip, nada. That means that protecting /64 prefixes need to be done by firewalling or to have ULA prefixes that are not routable. (ULA is like rfc1918 private IPv4 addresses).  
Having Tenant Networks mixed address space numberings add to complexity, so we'll go with firewalling. That has the added benefit that in a later phase we can add functionality to have multiple exit, as multiple routers as gateway is one of the features built in IPv6.


5. ### Considerations for 'Farms' (or nodes that live in a same subnet/prefix)

Nodes being part of a 'farm' and all of them deployed in the same Datacentre, connected to same switches, and mostly with multiple network interfaces are different beasts altogether. There we might envision to have specific containers that are using more performant interfaces.  
We can surmise that these are 'trusted' networks, if the user trusts the farmer (basically the same type of trust you can give to an AWS node, or a DigitalOcean Droplet). Given that, in order to keep an interconnect between sone customer workloads performant, and not using encryption for tunneling (also alleviating cpu usage, as encryption is either way an expensive thing to do), we can use basically the same concept but over vxlans or GRE tunnels (like all other players do).
