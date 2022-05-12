# Definition of words used throughout the documentation

## Node

  TL;DR: Computer.  
  A Node is a computer with CPU, Memory, Disks (or SSD's, NVMe) connected to _A_ network that has Internet access. (i.e. it can reach www.google.com, just like you on your phone, at home)  
  That Node will, once it has received an IP address (IPv4 or IPv6), register itself when it's new, or confirm it's identity and it's online-ness (for lack of a better word).

## TNo : Tenant Network object. [The gory details here](https://github.com/threefoldtech/zos/blob/master/modules/network.go)  

  TL;DR: The Network Description.  
  We named it so, because it is a data structure that describes the __whole__ network a user can request (or setup).  
  That network is a virtualized overlay network.  
  Basically that means that transfer of data in that network *always* is encrypted, protected from prying eyes, and __resources in that network can only communicate with each other__ **unless** there is a special rule that allows access. Be it by allowing access through firewall rules, *and/or* through a proxy (a service that forwards requests on behalf of, and ships replies back to the client).

## NR: Network Resource

  TL;DR: the Node-local part of a TNo.  
  The main building block of a TNo; i.e. each service of a user in a Node lives in an NR.  
  Each Node hosts User services, whatever type of service that is. Every service in that specific node will always be solely part of the Tenant's Network. (read that twice).  
  So: A Network Resource is the thing that interconnects all other network resources of the TN (Tenant Network), and provides routing/firewalling for these interconnects, including the default route to the BBI (Big Bad Internet), aka ExitPoint.  
  All User services that run in a Node are in some way or another connected to the Network Resource (NR), which will provide ip packet forwarding and firewalling to all other network resources (including the Exitpoint) of the TN (Tenant Network) of the user. (read that three times, and the last time, read it slowly and out loud)