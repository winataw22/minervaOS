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

5. Mark a node a being an exit node

The farmer then needs to select which node he agrees to use as an exit node for the grid

```bash
tffarmer select-exit kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=
#Node kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA= marked as exit node
```

## How to create a user private network

The only thing a user needs to do before creating a new private network is to select a farm with an exit node. Then he needs to do a request to the TNODB for a new network. The request is a POST request to the `/networks` endpoint of the TNODB with the body of the request containing the identity of the chosen exit farm.

```json
{"exit_farm": "ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E="}
```

The response body will contain a [network objet](https://github.com/threefoldtech/zosv2/blob/09de5a396bf60b794d2930ced1079a38bd5a9724/modules/network.go#L63). The network objet has an identifier, the network ID. The user can now use this network ID when he wants to provision some container on the grid.
