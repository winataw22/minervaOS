# Network module

## ZBus

Network module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| network|[network](#interface)| 0.0.1|

## Home Directory
network keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/network`|


## Interface

```go
//Networker is the interface for the network module
type Networker interface {
	ApplyNetResource(Network) (string, error)
	DeleteNetResource(Network) error
	Namespace(NetID) (string, error)
}
```

## Zero-OS networking

### Some First Explanations

Zero-OS is meant to provide services in the Threefold grid, and with grid, we naturally understand that the nodes (or their hosted services) need to be reachable for external users or for each other. So networking in 0-OS is a big thing, even when you assume that 'the network' is ubiquitous and always there, many things need to happen correctly before having a netWORK.  
For this, apart from all the other absolutely wonderful services in 0-OS, there is the network daemon. If it doesn't succeed it's bootstrap, nothing else will, and 0-OS will stop there.

So it (the network daemon, that is)
  - Configures the Node's initial network configuration, so that the Node can register itself.  For now we assume that the Node is connected to a network (ethernet segment) that provides IP addresses over DHCP, be it IPv4 or IPv6, or that there is a Routing Avertisement (RA) daemon for IPv6 running on that network.  
  Only once it has received an IP Address, most other internal services will be able to start. ([John Gage](https://www.networkcomputing.com/cloud-infrastructure/network-computer-again) from Sun said that `The Network is the Computer`, here that is absolutely true)

  - Notifies [zinit](https://github.com/threefoldtech/zinit/blob/master/docs/readme.md) (the services orchestrator in 0-OS) that it can register the dhcp client as a permanent process on the intitially discovered NIC (Network Interface Card) and that zinit can start other processes, one of which takes care of registering the node to the grid. (more elaborate explanation about that in [identity service](../identity/readme.md).

  - Listens in on the zbus for new or updated Network Resources (NR) that get sent by the provision daemon and applies them.

[Here some thought dumps from where we started working this out](../../specs/network/Requirements.md)

### Jargon

So. Let's have some abbreviations settled first:

  - #### Node : simple  
  TL;DR: Computer.  
  A Node is a computer with CPU, Memory, Disks (or SSD's, NVMe) connected to _A_ network that has Internet access. (i.e. it can reach www.google.com, just like you on your phone, at home)  
  That Node will, once it has received an IP address (IPv4 or IPv6), register itself when it's new, or confirm it's identity and it's online-ness (for lack of a better word).

  - #### TNo : Tenant Network object. [The gory details here](https://github.com/threefoldtech/zosv2/blob/master/modules/network.go)  
  TL;DR: The Network Description.  
  We named it so, because it is a datastructure that describes the __whole__ network a user can request (or setup).  
  That network is a virtualized overlay network.  
  Basically that means that transfer of data in that network *always* is encrypted, protected from prying eyes, and __resources in that network can only communicate with each other__ **unless** there is a special rule that allows access. Be it by allowing accesss through firewall rules, *and/or* through a proxy (a service that forwards requests on behalf of, and ships replies back to the client).

  - #### A Tno has an ExitPoint.  
  TL;DR: Any network needs to get out *somewhere*. [Some more explanation](exitpoints.md)  
  A Node that happens to live in an Internet Network (to differentiate from a Tenant network), more explictly, a network that is directly routable and accessible (unlike a home network), can be specified as an Exit Node.  
  That Node can then host Exitpoints for Tenant Networks.  
  Let's explain that.  
  Entities in a Tenant Network, where a TN being an overlay network, can only communicate with peers that are part of that network. At a certain point there is a gateway needed for this network to communicate with the 'external' world (BBI): that is an ExitPoint. ExitPoints can only live in Nodes designated for that purpose, namely Exit Nodes. Exit Nodes can only live in networks that are bidirectionally reachable for THE Internet (BBI).  
  An ExitPoint is *always* a part of a Network Resource (see below).

  - #### Network Resource: (NR)  
  TL;DR: the Node-local part of a TNo.  
  The main building block of a TNo; i.e. each service of a user in a Node lives in an NR.  
  Each Node hosts User services, whatever type of service that is. Every service in that specific node will always be solely part of the Tenant's Network. (read that twice).  
  So: A Network Resource is the thing that interconnects all other network resources of the TN (Tenant Network), and provides routing/firewalling for these interconnects, including the default route to the BBI (Big Bad Internet), aka ExitPoint.  
  All User services that run in a Node are in some way or another connected to the Network Resource (NR), which will provide ip packet forwarding and firewalling to all other network resources (including the Exitpoint) of the TN (Tenant Network) of the user. (read that three times, and the last time, read it slowly and out loud)

  -  #### IPAM IP Adress management
  TL;DR Give IP Adresses to containers attached to the NR's bridge.
  When the provisioner wants to start a container that doesn't attach itself to the NR's network namespace (cool that you can do that), but instead needs to create a veth pair and attach it to the NR's preconfigured bridge, the veth end in the container needs to get an IP address in the NR's Prefix (IPv6) and subnet (IPv4).  
  The NR has a deterministic IPv4 subnet definition that is coupled to the 7-8th byte of the IPv6 Prefix, where it then can use an IPv4 in the /24 CIDR that is assigned to the NR.
  As for the IPv6 address, you can choose to have a mac address derived IPv6 address, or/and a fixed address based on the same IPv4 address you gave to the container's interface.  
  Note: 
    - a veth pair is a concept in linux that creates 2 virtual network interfaces that are interconnected with a virtual cable. what goes in on one end of the pair, gets out on the other end, and vice-versa.
    - a bridge in linux is a concept of a virtual switch that can contain virtual interfaces. When you attach an interface to a bridge, it is a virtual switch with one port. You can add as many interfaces to that virtual switch as you like.





