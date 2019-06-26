### Node has public access through another path than it's 0-boot path

A Farmer that can provide vor ExitNodes will need to register these Exitnodes on the Grid as such, where:
  - He registers the IPv6 allocation that he received
  - He registers the IPv4 subnet 
  - Specifies which started an self-registered nodes are effectively ExitNodes
  - Has his switches and router configured to forward the correct Prefixes/Subnets to the environment

The Registry will hand out 

```
public obj:
    allocation:
        IPv4:
            IPv4 addr/mask
            IPv4 gateway
        IPv6:
            IPv6 addr/mask
            IPv6 gateway

get public obj from Registry




```
