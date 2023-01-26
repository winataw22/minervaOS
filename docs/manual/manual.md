# Manual
This document explain the usage of `ZOS`. `ZOS` usually pronounced (zero OS), got it's name from the idea of zero configuration. Since after the initial `minimal` configuration which only include which `farm` to join and what `network` (`development`, `testing`, or `production`) the owner of the node does not has to do anything more, and the node work fully autonomous.

The farmer himself cannot control the node, or access it by any mean. The only way you can interact with a node is via it's public API.

## Farm? Network? What are these?
Well, `zos` is built to allow people to run `workloads` around the world this simply is enabled by allowing 3rd party data-centers to run `ZOS` on their hardware. Then a user can then find any nearby `farm` (is what we call a cluster of nodes that belong to the same `farmer`) and then they can choose to deploy capacity on that node/farm. A `farm` can consist of one or more nodes.

So what is `network`.Well, to allow developers to build and `zos` itself and make it available during the early stages of development for testers and other enthusiastic people to try it out. To allow this we created 3 `networks`
- `development`: This is used mainly by developers to test their work. This is still available for users to deploy their capacity on (for really really cheap prices), but at the same time there is no grantee that it's stable or that data loss or corruption will happen. Also the entire network can be reset with no heads up.
- `testing`: Once new features are developed and well tested on `development` network they are released to `testing` environment. This also available for users to use with a slightly higher price than `development` network. But it's much more stable. In theory this network is stable, there should be no resets of the network, issues on this network usually are not fatal, but partial data loss can still occurs.
- `production`: Well, as the name indicates this is the most stable network (also full price) once new features are fully tested on `testing` network they are released on `production`.

## Creating a farm
While this is outside the scope of this document here you are a [link](https://library.threefold.me/info/manual/#/manual__create_farm)

# Interaction
`ZOS` provide a simple `API` that can be used to:
- Query node runtime information
  - Network information
    - Free `wireguard` ports
    - Get public configuration
  - System version
  - Other (check client for details)
- Deployment management (more on that later)
  - Create
  - Update
  - Delete

Note that `zos` API is available over `rmb` protocol. `rmb` which means `reliable message bus` is a simple messaging protocol that enables peer to peer communication over `yggdrasil` network. Please check [`rmb`](https://github.com/threefoldtech/rmb) for more information.

Simply put, `RMB` allows 2 entities two communicate securely knowing only their `id` an id is linked to a public key on the blockchain. Hence messages are verifiable via a signature.

To be able to contact the node directly you need to run
- `yggdrasil`
- `rmb` (correctly configured)

Once you have those running you can now contact the node over `rmb`. For a reference implementation (function names and parameters) please refer to [client](../../client/)

Here is a rough example of how low level creation of a deployment is done.

```go
cl, err := rmb.Default()
if err != nil {
	panic(err)
}
```
then create an instance of the node client
```go
node := client.NewNodeClient(NodeTwinID, cl)
```
define your deployment object
```go
dl := gridtypes.Deployment{
	Version: Version,
	TwinID:  Twin, //LocalTwin,
	// this contract id must match the one on substrate
	Workloads: []gridtypes.Workload{
		network(), // network workload definition
		zmount(), // zmount workload definition
		publicip(), // public ip definition
		zmachine(), // zmachine definition
	},
	SignatureRequirement: gridtypes.SignatureRequirement{
		WeightRequired: 1,
		Requests: []gridtypes.SignatureRequest{
			{
				TwinID: Twin,
				Weight: 1,
			},
		},
	},
}
```
compute hash
```go
hash, err := dl.ChallengeHash()
if err != nil {
	panic("failed to create hash")
}
fmt.Printf("Hash: %x\n", hash)
```
create the contract on `substrate` and get the `contract id` then you can link the deployment to the contract, then send to the node.

```go
dl.ContractID = 11 // from substrate
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err = node.DeploymentDeploy(ctx, dl)
if err != nil {
	panic(err)
}
```

Once the node receives the deployment. It will then fetch the contract (using the contract id) from the node recompute the deployment hash and compare with the one set on the contract. If matches, the node proceeds to process the deployment.

# Deployment
A deployment is a set of workloads that are contextually related. Workloads in the same deployment can reference to other workloads in the same deployment. But can't be referenced from another deployment. Well, except the network workload which can be referenced from a different deployment as long it belongs to the same user.

Workloads has unique IDs (per deployment) that are set by the user, hence he can create multiple workloads then reference to them with the given IDs (`names`)

For example, a deployment can define
- A private network with id `net`
- A disk with id `data`
- A public IP with id `ip`
- A container that uses:
  - The container can mount the disk like `mount: {data: /mount/path}`.
  - The container can get assign the public IP to itself like by referencing the IP with id `ip`.
  - etc.

## Workload
Each workload has a type which associated with some data. So minimal definition of a workload contains:
- `name`: unique per deployment (id)
- `type`: workload type
- `data`: workload data that is proper for the selected type.

The full workload structure is defined in this [file](../../pkg/gridtypes/workload.go) as
```go

// Workload struct
type Workload struct {
	// Version is version of reservation object. On deployment creation, version must be 0
	// then only workloads that need to be updated must match the version of the deployment object.
	// if a deployment update message is sent to a node it does the following:
	// - validate deployment version
	// - check workloads list, if a version is not matching the new deployment version, the workload is untouched
	// - if a workload version is same as deployment, the workload is "updated"
	// - if a workload is removed, the workload is deleted.
	Version uint32 `json:"version"`
	//Name is unique workload name per deployment  (required)
	Name Name `json:"name"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type WorkloadType `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data"`
	// Metadata is user specific meta attached to deployment, can be used to link this
	// deployment to other external systems for automation
	Metadata string `json:"metadata"`
	//Description human readale description of the workload
	Description string `json:"description"`
	// Result of reservation, set by the node
	Result Result `json:"result"`
}
```

### Types
- Virtual machine related
  - [`network`](network/readme.md)
  - [`ip`](ip/readme.md)
  - [`zmount`](zmount/readme.md)
  - [`zmachine`](zmachine/readme.md)
  - [`zlogs`](zlogs/readme.me)
- Storage related
  - [`zdb`](zdb/readme.md)
  - [`qsfs`](qsfs/readme.md)
- Gateway related
  - [`gateway-name-proxy`](gateway/name-proxy.md)
  - [`gateway-fqdn-proxy`]((gateway/fqdn-proxy.md))

### API
Node is always connected to the RMB network with the node `twin`. Means the node is always reachable over RMB with the node `twin-id` as an address.

The [node client](../../client/node.go) should have a complete list of all available functions. documentations of the API can be found [here](api.md)
