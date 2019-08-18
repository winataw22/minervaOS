
### A (tenant) network defines interconnected networks and their (single?) exit.

```rust
type network {
    // some type of ientification... an uuid ?
    // that netid is bound to a user and an allowed (bought) creation of a
    // node-local prefix for a bridge/container/vm
    // needs to be queried from somewhere(TBD) to be filled in
    netid
    // a netresource is a group of interconnected prefixes for a netid
    // needs to be queried and updated when the netresource is created
    []netresource 
    // the exit is the ultimate default gateway container
    // as well the prefix as the local config needs to be queried. 
    // - the prefix from the grid
    // - the exit prefix and default gw from the local allocation
    exit
}

type netresource {
    // where does it live
    nodeid
    // prefix is the IPv6 allocation that will be connected to the 
    // bridge/container/vm
    prefix
    // what are the peers:
    // each netresource needs to know what prefixes are reachable through
    // what endpoint. Basically this `connected` array will be used to build 
    // up the wireguard config in each netresource.
    []connected
    // a list of firewall rules to open access directly, IF that netresource
    // would be directly routed (future)
    Option<[]ipv6_allow>
}

// the exit container(ns) hold as well a prefix as netresource as well
// an IPv6 address that is going to hold the routes for all prefixes of the
// network. That IPv6 address will thus be the gateway for all prefixes that
// are part of that network. That also means that an upstream router needs to
// know the list of prefixes that need to be routed to that IPv6 address.
// An upstream router is the entry point toward nodes that have only IPv6 acces
// throug tunnels (like nodes in ipv4-only networks or home networks)
type exit {
    // netresource is the same as on all other netresources of a tenant network
    netresource
    // the ultimate IPv{4,6} config of the exit container.
    ipv4_conf
    []ipv4_dnat

    ipv6_conf
    []ipv6_allow
}
// TBD ipv4/6 portforwarding/firewalling

type ipv4_conf {
    // cidr
    ipaddress 
    Option<gateway>
    metric
    // deterministic name in function of the prefix and it's allocation
    interface
    // TBD, we need to establish if we want fc00/7 (ULA) or rfc1918 networks 
    // to be NATed (6to4 and/or 66)
    nat
}
// comments: see above
type ipv6_conf {
    ipaddress // cidr
    gateway
    metric
    interface
}

// connected are the peers for which we have a tunnel establised, and the
// prefix it routes to. The connection, as it is a peer to peer connection,
// can be of type wireguard, but also any other type that can bring
// a packet to a node containing a netresource.
// If for instance that node lives in the same subnet, it'll be a lot more
// efficient to set up a vxlan (multicast or with direct fdb entries), than
// using wireguard tunnels (that can be seen in a later phase)
type connected {
    connection
    prefix
}

// see above
enum connection {
    wireguard
    localvxlan
    
}

// see above for the type definition.
// the key would be a public key, with the private key only available
// locally and stored locally. 
type wireguard {
    // deterministic, based on the public key
    nicname
    // TBD, a peer can be IPv6, IPv6-ll or IPv4
    peer
    // public key
    key
    // Gateways in IPv6 are link-local. To be able to use IPv6 in any way,
    // an interface needs an IPv6 link-local address. As wireguard interfaces 
    // are l3-only, the kernel doesn't assign one, so we need to assign one
    // ourselves. (we need to come up with a deterministic way, so we can be
    // sure we now which/where)
    ll_fe80
}

// definition for later usage
// an l2vxlan wil be connected to a default bridge that gets attached to the
// network resource. That way we can easily add a vxlan to that bridge for 
// local interconnectivity
type l2vxlan {
    // deterministic or stored... 
    nicname
    // Or it's through fdb entries 
    Option<Vec<peer>>
    // Or it's in a multicast vxlan
    Option<group>
    // a vxlan always has an ID
    id
}

```

### Other things that need to be in place.

  - Infrastructure gateway nodes for hosting exit network resources.  
  For an IPv6 network to reach the internet, be it tunneled or in a local DC that has given out an allocation (/40, /48, /56), we need nodes that will :
    1. give out prefix allocations for network resources that get routed through their respective exit network resources
    1. set up the exit gateways and update routing tables for these network resources
  - IPv6 is a very different beast, where multihoming and mobile networks can be setup without ever needing to leave the stack. Basically, we could even envision 'standard' IPSec for our interlinks, if we're staying fully IPv6 on full IPv6 routed networks. where the keys would be IPSec pre-shared keys. That could also be a sloution in a multi-node farm within a same network.

### QUESTIONS:

  - Storage of network objects.  
  It has been said that we will use the 'blockchain' to store 'things'. A network/network resource could very well be such a 'thing', but at the moment there is no clear view on how to do that and how big these 'things' can be. Also, that 'thing' needs to be encrypted to make sure we don't enlarge a possible attack surface towards a tenant network. 
  
  - Allocation handouts for network resources and their respective exits.  
  There is a big difference between a node hosted in a home network or an IPv4-only network and a node that lives in a network that allows for multiple subnets. Case in point:  
    1. A node in an IPv4 Home network.  
    Network resources in that node will have an exit in a node that lives in a network that can hand out IPv6 allocations. Which also means that tenant networks are basically entities on their own, totally detached from the concept of a farmer selling a full stack resource. A farmer sells normally cpu,storage,memory, but there is no mention of Networking in most of our calculations.  
    But to alow for these resources to be available, that farmer will have to rely on Threefold or another farmer to have a network.
    1. A node in a farm that lives in an IPv4-only DC: same as above.
    1. A node in a farm in a DC or home that can hand out IPv6 allocations.  
    Something in that DC, or something that knows the alocations/farmer releationship has to hand out IPv6 allocations for requested network resources, that are not in use.
 
  - Notification of change of network objects to interested parties.  
  In case we can use the blockchain to store things, how will other nodes be notified a new network resource is created so that tunnels and routing can be built to establish connectivity to all network resources of a tenant network

  - Static routing  
  With all these network resources, static routing can get very hairy very fast. I still think a dynamic routing protocol will make life a lot easier. The advantage is also that routing protocols can detect unavailable links and update routes and/or circumvent the broken link.   
  That being said, there is still some research and testing to do on how to handle erroneous wg links, and redefine a route through another. Depending on the size of a tenant network, most standard peer-to-peer meshes will just be allright for now.
