# ZOSv2 network considerations

Running ZOS on a node is just a matter of booting it with a USB stick, or with a dhcp/bootp/tftp server with the right configuration so that the node can start the OS.
Once it starts booting, the OS detects the NICs, and starts the network configuration. A Node can only continue it's boot process till the end when it effectively has received an IP address and a route to the Internet. Without that, the Node will  retry indefinitely to obtain Internet access and not finish it's startup.

So a Node needs to be connected to a __wired__ network, providing a dhcp server and a default gateway to the Internet, be it NATed or plainly on the public network, where any route to the Internet, be it IPv4 or IPv6 or both is sufficient.

For a node to have that ability to host user networks, we **strongly** advise to have a working IPv6 setup, as that is the primary IP stack we're using for the User Network's Mesh to function.

## Running ZOS (v2) at home

Running a ZOS Node at home is plain simple. Connect it to your router, plug it in the network, insert the preconfigured USB stick containing the bootloader and the `farmer_id`, power it on.
You will then see it appear in the [Cockpit](https://cockpit.testnet.grid.tf/capacity), under your farm.

## Running ZOS (v2) in a multi-node farm in a DC

Multi-Node Farms, where a farmer wants to host the nodes in a data centre, have basically the same simplicity, but the nodes can boot from a boot server that provides for DHCP, and also delivers the iPXE image to load, without the need for a USB stick in every Node.

A boot server is not really necessary, but it helps ;-). That server has a list of the MAC addresses of the nodes, and delivers the bootloader over PXE. The farmer is responsible to set-up the network, and configure the boot server.

### Necessities

The Farmer needs to:

- Obtain an IPv6 prefix allocation from the provider. A `/64` will do, that is publicly reachable, but a `/48` is advisable if the farmer wants to provide IPv6 transit for User Networks
- If IPv6 is not an option, obtain an IPv4 subnet from the provider. At least one IPv4 address per node is needed, where all IP addresses are publicly reachable.
- Have the Nodes connected on that public network with a switch so that all Nodes are publicly reachable.
- In case of multiple NICS, also make sure his farm is properly registered in BCDB, so that the Node's public IP Addresses are registered.
- Properly list the MAC addresses of the Nodes, and configure the DHCP server to provide for an IP address, and in case of multiple NICs also provide for private IP addresses over DHCP per Node.
- Make sure that after first boot, the Nodes are reachable.

### IPv6

IPv6, although already a real protocol since '98, has seen reluctant adoption over the time it exists. That mostly because ISPs and Carriers were reluctant to deploy it, and not seeing the need since the advent of NAT and private IP space, giving the false impression of security.
But this month (10/2019), RIPE sent a mail to all it's LIRs that the last  consecutive /22 in IPv4 has been allocated. Needless to say, but that makes the transition to IPv6 in 2019 of utmost importance and necessity.
Hence, ZOS starts with IPv6, and IPv4 is merely an afterthought ;-)
So in a nutshell: we greatly encourage Farmers to have IPv6 on the Node's network.

### Routing/firewalling

Basically, the Nodes are self-protecting, in the sense that they provide no means at all to be accessed through listening processes at all. No service is active on the node itself, and User Networks function solely on an overlay.
That also means that there is no need for a Farm admin to protect the Nodes from exterior access, albeit some DDoS protection might be a good idea.
In the first phase we will still allow the Host OS (ZOS) to reply on ICMP ping requests, but that 'feature' might as well be blocked in the future, as once a Node is able to register itself, there is no real need to ever want to try to reach it.

### Multi-NIC Nodes

Nodes that Farmers deploy are typically multi-NIC Nodes, where one (typically a 1GBit NIC) can be used for getting a proper DHCP server running from where the Nodes can boot, and one other NIC (1Gbit or even 10GBit), that then is used for transfers of User Data, so that there is a clean separation, and possible injections bogus data is not possible.

That means that there would be two networks, either by different physical switches, or by port-based VLANs in the switch (if there is only one).

- Management NICs
  The Management NIC will be used by ZOS to boot, and register itself to the GRID. Also, all communications from the Node to the Grid happens from there.
- Public NICs

### Farmers and the grid

A Node, being part of the Grid, has no concept of 'Farmer'. The only relationship for a Node with a Farmer is the fact that that is registered 'somewhere (TM)', and that a such workloads on a Node will be remunerated with Tokens. For the rest, a Node is a wholly stand-alone thing that participates in the Grid.

```text
                                           172.16.1.0/24
                                           2a02:1807:1100:10::/64
+--------------------------------------+
| +--------------+                     |                    +-----------------------+
| |Node  ZOS     |             +-------+                    |                       |
| |              +-------------+1GBit  +--------------------+   1GBit switch        |
| |              | br-zos      +-------+                    |                       |
| |              |                     |                    |                       |
| |              |                     |                    |                       |
| |              |                     |                    +------------------+----+
| +--------------+                     |                                       |          +-----------+
|                                      |                      OOB Network      |          |           |
|                                      |                                       +----------+ ROUTER    |
|                                      |                                                  |           |
|                                      |                                                  |           |
|                                      |                                                  |           |
|                    +------------+    |                                       +----------+           |
|                    |  Public    |    |                                       |          |           |
|                    | container  |    |                                       |          +-----+-----+
|                    |            |    |                                       |                |
|                    |            |    |                                       |                |
|                    +---+--------+    |                   +-------------------+--------+       |
|                        |             |                   |  10GBit Switch             |       |
|                  br-pub|     +-------+                   |                            |       |
|                        +-----+10GBit +-------------------+                            |       +---------->
|                              +-------+                   |                            |        Internet
|                                      |                   |                            |
|                                      |                   +----------------------------+
+--------------------------------------+
                                          185.69.167.128/26        Public network
                                          2a02:1807:1100:0::/64

```

Where the underlay part of the wireguard interfaces get instantiated in the Public container (namespace), and once created these wireguard interfaces get sent into the User Network (Network Resource), where a user can then configure the interface a he sees fit.

The router of the farmer fulfills 2 roles:

- NAT everything in the OOB network to the outside, so that nodes can start and register themselves, as well get tasks to execute from the BCDB.
- Route the assigned IPv4 subnet and IPv6 public prefix on the public segment, to which the public container is connected.

As such, in case that the farmer wants to provide IPv4 public access for grid proxies, the node will need at least one (1) IPv4 address. It's free to the farmer to assign IPv4 addresses to only a part of the Nodes.
On the other hand, it is quite important to have a proper IPv6 setup, because things will work out better.

It's the Farmer's task to set up the Router and the switches.

In a simpler setup (small number of nodes for instance), the farmer could setup a single switch and make 2 port-based VLANs to separate OOB and Public, or even wit single-nic nodes, just put them directly on the public segment, but then he will have to provide a DHCP server on the Public network.
