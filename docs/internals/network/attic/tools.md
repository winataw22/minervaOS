# Network

- [How does a farmer configure a node as exit node](#How-does-a-farmer-configure-a-node-as-exit-node)
- [How to create a user private network](#How-to-create-a-user-private-network)

## How does a farmer configure a node as exit node

For the network of the grid to work properly, some of the nodes in the grid need to be configured as "exit nodes".  An "exit node" is a node that has a publicly accessible IP address and that is responsible routing IPv6 traffic, or proxy IPv4 traffic.

A farmer that wants to configure one of his nodes as "exit node", needs to register it in the TNODB. The node will then automatically detect it has been configured to be an exit node and do the necessary network configuration to start acting as one.

At the current state of the development, we have a [TNODB mock](../../tools/tnodb_mock) server and a [tffarmer CLI](../../tools/tffarm) tool that can be used to do these configuration.

Here is an example of how a farmer could register one of his node as "exit node":

1. Farmer needs to create its farm identity

```bash
tffarmer register --seed myfarm.seed "mytestfarm"
Farm registered successfully
Name: mytestfarm
Identity: ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E=
```

2. Boot your nodes with your farm identity specified in the kernel parameters.

Take that farm identity create at step 1 and boot your node with the kernel parameters `farmer_id=<identity>`

for your test farm that would be `farmer_id=ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E=`

Once the node is booted, it will automatically register itself as being part of your farm into the [TNODB](../../tools/tnodb_mock) server.

You can verify that you node registered itself properly by listing all the node from the TNODB by doing a GET request on the `/nodes` endpoints:

```bash
curl http://tnodb_addr/nodes
[{"node_id":"kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=","farm_id":"ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E=","Ifaces":[]}]
```

3. Farmer needs to specify its public allocation range to the TNODB

```bash
tffarmer give-alloc 2a02:2788:0000::/32 --seed myfarm.seed
prefix registered successfully
```

4. Configure the public interface of the exit node if needed

In this step the farmer will tell his node how it needs to connect to the public internet. This configuration depends on the farm network setup, this is why this is up to the farmer to provide the detail on how the node needs to configure itself.

In a first phase, we create the internet access in 2 ways:

- the node is fully public: you don't need to configure a public interface, you can skip this step
- the node has a management interface and a nic for public
    then `configure-public` is required, and the farmer has the public interface connected to a specific public segment with a router to the internet in front.

```bash
tffarmer configure-public --ip 172.20.0.2/24 --gw 172.20.0.1 --iface eth1 kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=
#public interface configured on node kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=
```

We still need to figure out a way to get the routes properly installed, we'll do static on the toplevel router for now to do a demo.

The node is now configured to be used as an exit node.

5. Mark a node as being an exit node

The farmer then needs to select which node he agrees to use as an exit node for the grid

```bash
tffarmer select-exit kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=
#Node kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA= marked as exit node
```

## How to create a user private network

1. Choose an exit node
2. Request an new allocation from the farm of the exit node
  - a GET request on the tnodb_mock at `/allocations/{farm_id}` will give you a new allocation
3. Creates the network schema

Steps 1 and 2 are easy enough to be done even manually but step 3 requires a deep knowledge of how networking works
as well as the specific requirement of 0-OS network system.
This is why we provide a tool that simplify this process for you, [tfuser](../../tools/tfuser).

Using tfuser creating a network becomes trivial:

```bash
# creates a new network with node DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk as exit node
# and output the result into network.json
tfuser generate --schema network.json network create --node DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk
```

network.json will now contains something like:

```json
{
  "id": "",
  "tenant": "",
  "reply-to": "",
  "type": "network",
  "data": {
    "network_id": "J1UHHAizuCU6s9jPax1i1TUhUEQzWkKiPhBA452RagEp",
    "resources": [
      {
        "node_id": {
          "id": "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk",
          "farmer_id": "7koUE4nRbdsqEbtUVBhx3qvRqF58gfeHGMRGJxjqwfZi",
          "reachability_v4": "public",
          "reachability_v6": "public"
        },
        "prefix": "2001:b:a:8ac6::/64",
        "link_local": "fe80::8ac6/64",
        "peers": [
          {
            "type": "wireguard",
            "prefix": "2001:b:a:8ac6::/64",
            "Connection": {
              "ip": "2a02:1802:5e::223",
              "port": 1600,
              "key": "PK1L7n+5Fo1znwD/Dt9lAupL19i7a6zzDopaEY7uOUE=",
              "private_key": "9220e4e29f0acbf3bd7ef500645b78ae64b688399eb0e9e4e7e803afc4dd72418a1c5196208cb147308d7faf1212758042f19f06f64bad6ffe1f5ed707142dc8cc0a67130b9124db521e3a65e4aee18a0abf00b6f57dd59829f59662"
            }
          }
        ],
        "exit_point": true
      }
    ],
    "prefix_zero": "2001:b:a::/64",
    "exit_point": {
      "ipv4_conf": null,
      "ipv4_dnat": null,
      "ipv6_conf": {
        "addr": "fe80::8ac6/64",
        "gateway": "fe80::1",
        "metric": 0,
        "iface": "public"
      },
      "ipv6_allow": []
    },
    "allocation_nr": 0,
    "version": 0
  }
}
```

Which is a valid network schema. This network only contains a single exit node though, so not really useful.
Let's add another node to the network:

```bash
tfuser generate --schema network.json network add-node --node 4hpUjrbYS4YeFbvLoeSR8LGJKVkB97JyS83UEhFUU3S4
```

result looks like:

```json
{
  "id": "",
  "tenant": "",
  "reply-to": "",
  "type": "network",
  "data": {
    "network_id": "J1UHHAizuCU6s9jPax1i1TUhUEQzWkKiPhBA452RagEp",
    "resources": [
      {
        "node_id": {
          "id": "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk",
          "farmer_id": "7koUE4nRbdsqEbtUVBhx3qvRqF58gfeHGMRGJxjqwfZi",
          "reachability_v4": "public",
          "reachability_v6": "public"
        },
        "prefix": "2001:b:a:8ac6::/64",
        "link_local": "fe80::8ac6/64",
        "peers": [
          {
            "type": "wireguard",
            "prefix": "2001:b:a:8ac6::/64",
            "Connection": {
              "ip": "2a02:1802:5e::223",
              "port": 1600,
              "key": "PK1L7n+5Fo1znwD/Dt9lAupL19i7a6zzDopaEY7uOUE=",
              "private_key": "9220e4e29f0acbf3bd7ef500645b78ae64b688399eb0e9e4e7e803afc4dd72418a1c5196208cb147308d7faf1212758042f19f06f64bad6ffe1f5ed707142dc8cc0a67130b9124db521e3a65e4aee18a0abf00b6f57dd59829f59662"
            }
          },
          {
            "type": "wireguard",
            "prefix": "2001:b:a:b744::/64",
            "Connection": {
              "ip": "<nil>",
              "port": 0,
              "key": "3auHJw3XHFBiaI34C9pB/rmbomW3yQlItLD4YSzRvwc=",
              "private_key": "96dc64ff11d05e8860272b91bf09d52d306b8ad71e5c010c0ccbcc8d8d8f602c57a30e786d0299731b86908382e4ea5a82f15b41ebe6ce09a61cfb8373d2024c55786be3ecad21fe0ee100339b5fa904961fbbbd25699198c1da86c5"
            }
          }
        ],
        "exit_point": true
      },
      {
        "node_id": {
          "id": "4hpUjrbYS4YeFbvLoeSR8LGJKVkB97JyS83UEhFUU3S4",
          "farmer_id": "7koUE4nRbdsqEbtUVBhx3qvRqF58gfeHGMRGJxjqwfZi",
          "reachability_v4": "hidden",
          "reachability_v6": "hidden"
        },
        "prefix": "2001:b:a:b744::/64",
        "link_local": "fe80::b744/64",
        "peers": [
          {
            "type": "wireguard",
            "prefix": "2001:b:a:8ac6::/64",
            "Connection": {
              "ip": "2a02:1802:5e::223",
              "port": 1600,
              "key": "PK1L7n+5Fo1znwD/Dt9lAupL19i7a6zzDopaEY7uOUE=",
              "private_key": "9220e4e29f0acbf3bd7ef500645b78ae64b688399eb0e9e4e7e803afc4dd72418a1c5196208cb147308d7faf1212758042f19f06f64bad6ffe1f5ed707142dc8cc0a67130b9124db521e3a65e4aee18a0abf00b6f57dd59829f59662"
            }
          },
          {
            "type": "wireguard",
            "prefix": "2001:b:a:b744::/64",
            "Connection": {
              "ip": "<nil>",
              "port": 0,
              "key": "3auHJw3XHFBiaI34C9pB/rmbomW3yQlItLD4YSzRvwc=",
              "private_key": "96dc64ff11d05e8860272b91bf09d52d306b8ad71e5c010c0ccbcc8d8d8f602c57a30e786d0299731b86908382e4ea5a82f15b41ebe6ce09a61cfb8373d2024c55786be3ecad21fe0ee100339b5fa904961fbbbd25699198c1da86c5"
            }
          }
        ],
        "exit_point": false
      }
    ],
    "prefix_zero": "2001:b:a::/64",
    "exit_point": {
      "ipv4_conf": null,
      "ipv4_dnat": null,
      "ipv6_conf": {
        "addr": "fe80::8ac6/64",
        "gateway": "fe80::1",
        "metric": 0,
        "iface": "public"
      },
      "ipv6_allow": []
    },
    "allocation_nr": 0,
    "version": 1
  }
}
```

Our network schema is now ready, but before we can provision it onto a node, we need to sign it and send it to the bcdb.
To be able to sign it we need to have a pair of key. You can use `tfuser id` command to create an identity:

```bash
tfuser id --output user.seed
```

We can now provision the network on both nodes:

```bash
tfuser provision --schema network.json \
--node  DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk \
--node 4hpUjrbYS4YeFbvLoeSR8LGJKVkB97JyS83UEhFUU3S4 \
--seed user.seed
```
