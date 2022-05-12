# 0-OS v2 and it's network

## Introduction

0-OS nodes participating in the Threefold grid, need connectivity of course. They need to be able to communicate over 
the Internet with each-other in order to do various things:

- download it's OS modules
- perform OS module upgrades
- register itself to the grid, and send regular updates about it's status
- query the grid for tasks to execute
- build and run the Overlay Network
- download flists and the effective files to cache

The nodes themselves can have connectivity in a few different ways:

- Only have RFC1918 private addresses, connected to the Internet through NAT, NO IPv6
  Mostly, these are single-NIC (Network card) machines that can host some workloads through the Overlay Network, but 
  cant't expose services directly. These are HIDDEN nodes, and are mostly booted with an USB stick from 
  bootstrap.grid.tf .
- Dual-stacked: having RFC1918 private IPv4 and public IPv6 , where the IPv6 addresses are received from a home router, 
but firewalled for outgoing traffic only. These nodes are effectively also HIDDEN
- Nodes with 2 NICs, one that has effectively a NIC connected to a segment that has real public 
addresses (IPv4 and/or IPv6) and one NIC that is used for booting and local 
management. (OOB) (like in the drawing for farmer setup)

For Farmers, we need to have Nodes to be reachable over IPv6, so that the nodes can:

- expose services to be proxied into containers/vms
- act as aggregating nodes for Overlay Networks for HIDDEN Nodes

Some Nodes in Farms should also have a publicly reachable IPv4, to make sure that clients that only have IPv4 can 
effectively reach exposed services.

But we need to stress the importance of IPv6 availability when you're running a multi-node farm in a datacentre: as the 
grid is boldly claiming to be a new Internet, we should make sure we adhere to the new protocols that are future-proof. 
Hence: IPv6 is the base, and IPv4 is just there to accomodate the transition.

Nowadays, RIPE can't even hand out consecutive /22 IPv4 blocks any more for new LIRs, so you'll be bound to market to 
get IPv4, mostly at rates of 10-15 Euro per IP. Things tend to get costly that way.

So anyway, IPv6 is not an afterthought in 0-OS, we're starting with it.

## Physical setup for farmers

```text
                      XXXXX  XXX
                     XX   XXX  XXXXX XXX
                    X           X      XXX
                   X                     X
                   X      INTERNET       X
                   XXX        X          X
                     XXXXX   XX    XX XXXX
                      +X XXXX  XX XXXXX
                      |
                      |
                      |
                      |
                      |
               +------+--------+
               | FIREWALL/     |
               | ROUTER        |
               +--+----------+-+
                  |          |
      +-----------+----+   +-+--------------+
      |  switch/       |   |  switch/       |
      |  vlan segment  |   |  vlan segment  |
      +-+---------+----+   +---+------------+
        |         |            |
+-------+-------+ |OOB         | PUBLIC
| PXE / dhcp    | |            |
| Ser^er        | |            |
+---------------+ |            |
                  |            |
            +-----+------------+----------+
            |                             |
            |                             +--+
            |                             |  |
            |  NODES                      |  +--+
            +--+--------------------------+  |  |
               |                             |  |
               +--+--------------------------+  |
                  |                             |
                  +-----------------------------+
```

The PXE/dhcp can also be done by the firewall, your mileage may vary.

## Switch and firewall configs

Single switch, multiple switch, it all boils down to the same:

- one port is an access port on an OOB vlan/segment
- one port is connected to a public vlan/segment

The farmer makes sure that every node receives properly an IPv4 address in the OOB segment through means of dhcp, so 
that with a PXE config or USB, a node can effectively start it's boot process:

- Download kernel and initrd
- Download and mount the system flists so that the 0-OS daemons can start
- Register itself on the grid
- Query the grid for tasks to execute

For the PUBLIC side of the Nodes, there are a few things to consider:

- It's the farmer's job to inform the grid what node gets an IP address, be it IPv4 or IPv4.
- Nodes that don't receive and IPv4 address will connect to the IPv4 net through the NATed OOB network
- A farmer is responsible to provide and IPv6 prefix on at least one segment, and have a Router Advertisement daemon 
runnig to provide for SLAAC addressin on that segment.
- That IPv6 Prefix on the public segment should not be firewalled, as it's impossible to know in your firewall what 
ports will get exposed for the proxies.

The Nodes themselves have nothing listening that points into the host OS itself, and are by themselves also firewalled. 
In dev mode, there is an ssh server with a key-only login, accessible by a select few ;-)

## DHCP/Radvd/RA/DHCP6

For home networks, there is not much to do, a Node will get an IPv4 Private(rfc1918) address , and most probaly and 
ipv6 address in a /64 prefix, but is not reachable over ipv6, unless the firewall is disabled for IPv6. As we can't 
rely on the fact that that is possible, we assume these nodes to be HIDDEN.

A normal self-respecting Firewall or IP-capable switch can hand out IP[46] addresses, some can 
even bootp/tftp to get nodes booted over the network.
We are (full of hope) assuming that you would have such a beast to configure and splice your network 
in multiple segments.  
A segment is a physical network separation. That can be port-based vlans, or even separate switches, whatver rocks your 
boat, the keyword is here **separate**.

On both segments you will need a way to hand out IPv4 addresses based on MAC addresses of the nodes. Yes, there is some 
administration to do, but it's a one-off, and really necessary, because you really need to know whic physical machine 
has which IP. For lights-out management and location of machines that is a must.

So you'll need a list of mac addresses to add to your dhcp server for IPv4, to make sure you know which machine has 
received what IPv4 Address.
That is necessary for 2 things:

- locate the node if something is amiss, like be able to pinpoint a node's disk in case it broke (which it will)
- have the node be reachable all the time, without the need to update the grid and network configs every time the node 
boots.

## What happens under the hood (farmer)

While we did our uttermost best to keep IPv4 address needs to a strict minimum, at least one Node will need an IPv4 address for handling everything that is Overlay Networks.  
For Containers to reach the Internet, any type of connectivity will do, be it NAT or though an Internal DMZ that has a 
routable IPv4 address.

Internally, a lot of things are being set-up to have a node properly participate in the grid, as well to be prepared to partake in the User's Overlay Networks.

A node connects itself to 'the Internet' depending on a few states.

1. It lives in a fully private network (like it would be connected directly to a port on a home router)

```
      XX  XXX
    XXX     XXXXXX
   X  Internet   X
   XXXXXXX   XXXXX
       XX  XXX
      XX   X
         X+X
          |
          |
 +--------+-----------+
 | HOME /             |
 | SOHO router        |
 |                    |
 +--------+-----------+
          |
          |  Private space IPv4
          |  (192.168.1.0/24)
          |
+---------+------------+
|                      |
|  NODE                |
|                      |
|                      |
|                      |
|                      |
|                      |
+----------------------+
```

1. It lives in a fully public network (like it is connected directly to an uplink and has a public ipv4 address)

```
      XX  XXX
    XXX     XXXXXX
   X  Internet   X
   XXXXXXX   XXXXX
       XX  XXX
      XX   X
         X+X
          |
          | fully public space ipv4/6
          | 185.69.166.0/24
          | 2a02:1802:5e:0:1000::abcd/64
          |
+---------+------------+
|                      |
|  NODE                |
|                      |
+----------------------+

```
The node is fully reachable

1. It lives in a datacentre, where a farmer manages the Network.

A little Drawing :

```text
+----------------------------------------------------+
|  switch                                            |
|                                                    |
|                                                    |
+----------+-------------------------------------+---+
           |                                     |
  access   |                                     |
  mgmt     |                     +---------------+
  vlan     |                     | access
           |                     | public
           |                     | vlan
           |                     |
   +-------+---------------------+------+
   |                                    |
   |     nic1                  nic2     |
   |                                    |
   |                                    |
   |                                    |
   |   NODE                             |
   |                                    |
   |                                    |
   |                                    |
   +------------------------------------+

```

Or the more elaborate drawing on top that should be sufficient for a sysadmin to comprehend.

Although:

- we don't (yet) support nic bonding (next release)
- we don't (yet) support vlans, so your ports on switch/router need to be access ports to vlans to your router/firewall


## yeayea, but really ... what now ?

Ok, what are the constraints?

A little foreword:
ZosV2 uses IPv6 as it's base for networking, where the oldie IPv4 is merely an afterthought. So for it to work properly in it's actual incantation (we are working to get it to do IPv4-only too), for now, we need the node to live in a space that provides IPv6 __too__ .  
IPV4 and IPv6 are very different beasts, so any machine connected to the Internet wil do both on the same network. So basically your computer talks 2 different languages, when it comes to communicating. That is the same for ZOS, where right now, it's mother tongue is IPv6.

So your zos for V2 can start in different settings
1) you are a farmer, your ISP can provide you with IPv6
Ok, you're all set, aside from a public IPv4 DHCP, you need to run a Stateless-Only SLAAC Router Advertiser (ZOS does NOT do DHCP6).

1) you are a farmer, your ISP asks you what the hell IPv6 is
That is problematic right now, wait for the next release of ZosV2

1) you are a farmer, with only one node , at home, and on your PC https://ipv6.net tells you you have IPv6 on your PC.
That means your home router received an IPV6 allocation from the ISP, 
Your'e all set, your node will boot, and register to the grid. If you know what you're doing, you can configure your router to allow all ipv6 traffic in forwarding mode to the specifice mac address of your node. (we'll explain later)
1) you are a farmer, with a few nodes somewhere that are registered on the grid in V1, but you have no clue if IPv6 is supported where these nodes live 
1) you have a ThreefoldToken node at home, and still do not have a clue

Basically it boils down also in a few other cases

1) the physical network where a node lives has: IPv6 and Private space IPv4 
1) the physical network where a node lives has: IPv6 and Public IPv4
1) the physical network where a node lives has: only IPv4

But it bloils down to : call your ISP, ask for IPv6. It's the future, for yout ISP, it's time. There is no way to circumvent it. No way.


OK, then, now what.

1) you're a farmer with a bunch of nodes somewhere in a DC

  - your nodes are connected once (with one NIC) to a switch/router  
  Then your router will have :
    - a segment that carries IPv4 __and__ IPv6:

    - for IPv4, there are 2 possibilities:
      - it's RFC1918 (Private space) -> you NAT that subnet (e.g. 192.168.1.0/24) towards the Public Internet 
      
        - you __will__ have difficulty to designate a IPv4 public entrypoint into your farm
        - your workloads will be only reachable through the overlay
        - your storage will not be reachable
        
      - you received a (small, because of the scarceness of IPv4 addresses, your ISP will give you only limited and pricy IPv4 adresses) IPv4 range you can utilise

        - things are better, the nodes can live in public ipv4 space, where they can be used as entrypoint
        - standard configuration that works

    - for IPv6, your router is a Routing advertiser that provides SLAAC (Stateless, unmanaged) for that segment, working witha /64 prefix

      - the nodes will reachable over IPv6
      - storage backend will be available for the full grid
      - everything will just work
      
      Best solution for single NIC: 
        - an ipv6 prefx
        - an ipv4 subnet (however small)

  - your nodes have 2 connections, and you wnat to differ management from user traffic
  
    - same applies as above, where the best outcome will be obtained with a real IPv6 prefix allocation and a small public subnet that is routable.
      - the second NIC (typically 10GBit) will then carry everything public, and the first nic will just be there for managent, living in Private space for IPv4, mostly without IPv6 
      - your switch needs to be configured to provide port-based vlans, so the segments are properly separated, and your router needs to reflect that vlan config so that separation is handeled by the firewall in the router (iptables, pf, acl, ...)


  


