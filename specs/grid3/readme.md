# Introduction
This document is about tf-grid 3.0. It shows an overview of the grid regarding operation, and how different components communicate to each other.
## Definitions
- 3node: a machine that runs zos operating system.
- RMB: reliable message bus
- grid-db: a decentralized block chain database that allows nodes and other twins to share trusted data. Anyone can look up nodes and verify their identity, find their corresponding twin IDs to communicate over RMB.

# Overview
![Overlay](png/grid3-overlay.png)

The operation can be described as follows:
- Everything that needs to talk to other components should live on the yggdrasil network.
- Nodes and users have to create a “twin” object on GridDB which is associated with an Yggdrasil IP. Then to communicate with any twin, the IP can be looked up using the twin id. This is basically how RMB works.
- When starting for the first time, the node needs to register itself on the grid database, which is a decentralized database built on top of substrate. The registration need to have information about:
  - Which farm it belongs to
  - What capacity it has
  - Twin ID for that node (which is associated with its Yggdrasil IP)
  - Public Config
- Once identity has been established, secure and trusted communication can be established between the different parties.
- For a user to deploy a workload he need to do the following:
  - Create a **contract** on substrate
    - The **contract** is described in more details [here](contract.md)
    - It also has required number of public IPs to be reserved from the farm (in case you have public ips in your deployment definition)
  - Substrate needs to validate the contract, user, funds and lock up required public ips from the farm object.
  - Once a contract is created, the contract **id + deployment** is sent to the node.
  - The node then can read the contract data, validate and apply them. Deployment status can then be queried over RMB directly from the node.
  - Node will also send consumption reports to the contract, the contract then can start billing the user.

![Sequence Diagram](png/sequence.png)
# 3node
On first boot the node needs to create a “twin” on the grid, a twin associated with a public key. Hence it can create verifiable signed messages.
Then, a node will then register itself as a “node” on the grid, and links this twin to the node object.

Once the node is completely up, a user can reach out to the node to submit workload requests.
Responses from the node **MUST** be signed with the twin secret key, a user then can verify it against the node public key.

Node uses the same verification mechanism against requests from twins.

# Concerns
- Stellar - Substrate token bridge. How should we do this? Transaction on the grid db will cost tokens (economic protection)
- Deployments are not migrateable from a node to another. If a node is not reachable anymore or down. It’s up to the owner to delete his contract and recreate it somewhere else.

# Grid DB
## Definitions
- `entity`: this represents a legal entity or a person, the entity is the public key of the user associated with name, country, and a unique identifier.
- `twin`: represents the management interface that can be accessed over the yggdrasil ipv6 address. A twin is associated also with a single public key.

On the grid, we build on top of the above concepts a more sophisticated logics to represent the following: (note, full types specifications can be found here)
- farm: a farm is associated
  - entity-id: this defines the entity of the farm itself, so it holds information about the country, city, etc and public key.
  - twin-id: the communication endpoint for this farm.
- node:  associated with
  - entity-id: defines the entity of the node itself, (pubkey, city, country, etc…)
  - twin-id: defines the peer information (peer-id and public ipv6)
  - farm-id: unique farm id this node is part of.

Grid database is a decentralized blockchain database that leverage on substrate to provide API that can be used to
- Create Entities
- Create Twins
- Add / Remove entities from twins
- Create Nodes
- Create Farms
- Create Pricing policies for farms
- Create Certification Codes
- Query information about  entities (find node, or verify identities)

A farm has one twin, through this twin the peer_id and entity can be requested. This is also the case for nodes as they have one farm.

# Public IPs
Public IPs are locked by substrate from the farmer IPs pool. for this to be possible, on contract creation the user need to specify how many IPs he needs to lock up.

If the contract creation is successful. It means the contract managed to look up the required number of Ips. the values for the IPs are going to be available for this specific node.

Once the contract is terminated, the Ips are returned back to the farmer pool.

# Pricing
Each farmer object is assigned a Pricing Policy object:
The pricing policy defines:
- Currency (TFT, USD, etc…) _we will probably drop that_
- Unit (Bytes, Kilobyte, Megabyte, Gigabytes)
- SU
- CU
- NU

## General notes:
- Each price defines a price for a single UNIT/SECOND. So for example SU is the price of a single Storage Unit per second where a Storage Unit can be 1 Gigabyte
- NU is an exception because that's the price PER unit (regardless of number of seconds). For example if NU is 10 and Unit ig Gigabyte, so 1 Gigabyte of network traffic costs 10 mill.
- Price is defined as `mil` of the currency. 1 UNIT = 1,000,000,0 mil

### Example:
Currency: TFT
SU: 1000 (mil tft)

(Price for 1 Gigabyte of ssd storage costs 1000 / 10 000 000  TFT per second)

So let's assume capacity is created at **Time = T**
- Node send first report at **T+s** with SU consumption C = (10 gigabytes)
- price = C/(1024*1024*1024) * s * SU
- price = 10 * s * SU
- Assume s is 5 min (300 seconds)
- Then price = 10 * 300 * 1000 = 3000000 mil = 3 TFT

Same for each other unit EXCEPT the NU. The NU unit is incremental. It means the node will keep reporting the total amount of bytes consumed since the machine starts. So to correctly calculate the consumption over period S it has to be `now.NU - last.NU`
