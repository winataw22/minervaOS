# 0-OS v2 and it's network setup

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

## Network setup for farmers

This is a quick manual to what is needed for connecting a node with zero-OS V2.0

### Step 1. Testing for IPv6 availability in your location 
As descibed above the network in which the node is instaleld has to be IPv6 enabled.  This is not an afterthought as we are building a new internet it has to ba based on the new and forward looking IP addressing scheme.  This is something you have to investigate, negotiate with you connectivity provider.  Many (but not all home connectivity products and certainly most datacenters can provide you with IPv6.  There are many sources of infromation on how to test and check whether your connection is IPv6 enabled, [here is a starting point](http://www.ipv6enabled.org/ipv6_enabled/ipv6_enable.php)

### Step 2. Choosing you setup for connecitng you nodes.

Once you have established that you have IPv6 enabled on the network you are about to deploy, you have to make sure that there is an IPv6 DHCP facility available.  Zero-OS does not work with static IPv6 addresses (at this point in time).  So you have choose and create one of the following setups:

#### 2.1 Home setup

Use your (home) ISP router Ipv6 DHCP capabilities to provide (private) IPv6 addresses.  The principle will work the same as for IPv4 home connections, everything happens enabled by Network Adress Translation (just like anything else that uses internet connectivity).  This should be relatively straightforward if you have established that your conenction has IPv6 enabled.

#### 2.2 Datacenter / Expert setup

In this situation there are many options on how to setup you node.  This requires you as the expert to make a few decisions on how to connect what what the best setup is that you can support for the operaitonal time of your farm.  The same basics principles apply:
  - You have to have a block of (public) IPv6 routed to you router, or you have to have your router setup to provide Network Address Translation (NAT)
  - You have to have a DHCP server in your network that manages and controls IPV6 ip adress leases.  Depending on your specific setup you have this DHCP server manage a public IPv6y range which makes all nodes directly connected to the public internet or you have this DHCP server manage a private block og IPv6 addresses which makes all you nodes connect to the internet through NAT.  

As a farmer you are in charge of selecting and creating the appropriate network setup for your farm.  

## General notes

The above setup will allows your node(s) to appear in explorer on the TF Grid and will allowd you to earn farming tokens.  At stated in the introduction ThreeFold is creating next generation internet capacity and therefore has IPv6 as it's base building block.  Connecting to the current (dominant) IPv4 network happens for IT workloads through so called webgateways.  As the word sais these are gateways that provide connectivity between the currenct leading IPv4 adressing scheme and IPv6. 

We have started a forum where people share their experiences and configurations.  This will be work in progress and forever growing.

**IMPORTANT**:  You as a farmer do  not need access to IPV4 to be able to rent capacity for IT workloads that need to be visible on IPV4, this is something that can happen elswhere on the TF Grid.
 
