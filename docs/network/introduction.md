# Introduction to networkd the network manager of 0-OS

## Boot and initial setup

At boot, be it from an usb stick or PXE, ZOS starts up the kernel, with a few necessary parameters like farm ID and/or possible network parameters, but basically once the kernel has started, [zinit](https://github.com/threefoldtech/zinit) among other things, starts the network initializer.

In short, that process loops over the available network interfaces and tries to obtain an IP address that also provides for a default gateway. That means: it tries to get Internet connectivity. Without it, ZOS stops there, as not being able to register itself, nor start other processes, there wouldn't be any use for it to be started anyway.

Once it has obtained Internet connectivity, ZOS can then proceed to make itself known to the Grid, and acknowledge it's existence. It will then regularly poll the Grid for tasks.

Once initialized, with the network daemon running (a process that will handle all things related to networking), ZOS will set up some basic services so that workloads can themselves use that network.

## Networkd functionality

The network daemon is in itself responsible for a few tasks, and working together with the [provision daemon](../provision) it mainly sets up the local infrastructure to get the user network resources, together with the wireguard configurations for the user's mesh network.

The Wireguard mesh is an overlay network. That means that traffic of that network is encrypted and encapsulated in a new traffic frame that the gets transferred over the underlay network, here in essence the network that has been set up during boot of the node.

For users or workloads that run on top of the mesh, the mesh network looks and behaves like any other directly connected workload, and as such that workload can reach other workloads or services in that mesh with the added advantage that that traffic is encrypted, protecting services and communications over that mesh from too curious eyes.

That also means that workloads between nodes in a local network of a farmer is even protected from the farmer himself, in essence protecting the user from the farmer in case that farmer could become too curious.

As the nodes do not have any way to be accessed, be it over the underlaying network  or even the local console of the node, a user can be sure that his workload cannot be snooped upon.

## Techie talk

- **boot and initial setup**  
For ZOS to work at all (the network is the computer), it needs an internet connection. That is: it needs to be able to communicate with the BCDB over the internet.  
So ZOS starts with that: with the `internet` process, that tries go get the node to receive an IP address. That process will have set-up a bridge (`zos`), connected to an interface that is on an Internet-capable network. That bridge will have an IP address that has Internet access.
Also, that bridge is there for future public interfaces into workloads.  
Once ZOS can reach the Internet, the rest of the system can be  started, where ultimately, the `networkd` daemon is started.

- **networkd initial setup**  
`networkd` starts with recensing the available Network interfaces, and registers them to the BCDB (grid database), so that farmers can specify non-standard configs like for multi-nic machines. Once that is done, `networkd` registers itself to the zbus, so it can receive tasks to execute from the provsioning daemon (`provisiond`).  
These tasks are mostly setting up network resources for users, where a network resource is a subnet in the user's wireguard mesh.

- **multi-nic setups**  

When someone is a farmer, exploiting nodes somewhere in a datacentre, where the nodes have multiple NICs, it is advisable (though not necessary) to differentiate OOB traffic (like initial boot setup) from user traffic (as well the overlay network as the outgoing NAT for nodes for IPv4) to be on a different NIC. With these parameters, a user will have to make sure their switches are properly configured, more in docs later.

- **registering and configurations**  

Once a node has booted and properly initialized, registering and configuring the node to be able to accept workloads and their associated network configs, is a two-step process.  
First, the node registers it's live network setup to the BCDB. That is : all NICs with their associated IP addresses and routes are registered so a farm admin can in a second phase configure eventual separate NICs to handle different kinds of workloads.
In that secondary phase, a farm admin can then set-up the NICs and their associated IP's manually, so that workloads can start using them.

## Wireguard explanations

- **wireguard as pointopoint links and what that means**  
Wireguard is a special type of VPN, where every instance is as well server for multiple peers as client towards multiple peers. That way you can create fanning-out connections als receive connections from multiple peers, creating effectively a mesh of connections Like this : ![like so](HIDDEN-PUBLIC.png)

- **wireguard port management**  
Every wireguard point (a network resource point) needs a destination/port combo when it's  publicly reachable. The destination is a public ip, but the port is the differentiator. So we need to make sure every network wireguard listening port is unique in the node where it runs, and can be reapplied in case of a node's reboot.
ZOS registers the ports **already in use** to the BCDB, so a user can the pick a port that is not yet used.

- **wireguard and hidden nodes**  
Hidden nodes are nodes that are in essence hidden behind a firewall, and unreachable from the Internet to an internal network, be it as an IPv4 NATed host or an IPv6 host that is firewalled in any way, where it's impossible to have connection initiations form the Internet to the node.  
As such, these nodes can only partake in a network as client-only towards publicly reachable peers, and can only initiate the connections themselves. (ref previous drawing).  
To make sure connectivity stays up, the clients (all) have a keepalive towards all their peers so that communications towards network resources in hidden nodes can be established.

## Caveats

- **hidden nodes**  
Hidden nodes live (mostly) behind firewalls that keep state about connections and these states have a lifetime. We try at best to keep these communications going, but depending of the firewall your mileage may vary (YMMV ;-))

- **local underlay network reachability**  
When multiple nodes live in a same hidden network, at the moment we don't try to have the nodes establish connectivity between themselves, so all nodes in that hidden network can only reach each other through the intermediary of a node that is publicly reachable. So to get some performance, a farmer will have to have real routable nodes available in the vicinity.
So for now, a farmer is better off to have his nodes really reachable over a public network.

- **IPv6 and IPv4 considerations**  
While the mesh can work over IPv4 __and__ IPv6 at the same time, the peers can only be reached through one protocol at the same time. That is a peer is IPv4 __or__ IPv6, not both. Hence if a peer is reachable over IPv4, the client towards that peer needs to reach it over IPv4 too and thus needs an IPv4 address.
We advise strongly to have all nodes properly set-up on a routable unfirewalled IPv6 network, so that these problems have no reason to exist.
