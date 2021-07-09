# Deployment contract
The deployment contract is a contract between:
- User (owner of deployment)
- Node (Zos node)

The contract must satisfy the following requirements

- The user must be able to "reconstruct" his deployment. He (at any time) should be able to read the history of hist deployments, and reconstruct the full setup from the blockchain (substrate)
- The user information in the contract must be private to the user, even when the blockchain is public, only the user can read the "content" of this contract.
- The node must be able to validate the contract


# Proposal #1
This assumes the following constrains:
- Nodes work solo. A node only is concerned about itself, and doesn't know or care about other nodes. This is how they implemented right now and this simplify the node life and makes it much easier to manage. A complex multi node setup is orchestrated by an external tool on the client side.
- A single contract is between a **single** user and a **single** node. A multi node setup is orchestrated by the user, and the user need to create multiple contracts for each node involved in the development (this is to simplify the setup, a user can then read all his contracts and reconstruct his setup when needed)
- _OPTIONAL_: Single contract for multiple nodes probably can be implemented but will make implementing validation way more complex.

## Implementation
- Deployment: is a description of the entire deployment on a single node (network, volumes, vms, public_ips etc...). Please check the deployment structure [here](../../pkg/gridtypes/deployment.go)

- The user then create a contract as follows:
```js
contract = {
    // address is the node address.
    address: "<node address>"
    // data is the encrypted deployment body. This encrypted the deployment with the **USER** public key. So only the user can read this data later on (or any other key that he keeps safe).
    // this data part is read only by the user and can actually hold any information to help him reconstruct his deployment or can be left empty.
    data: encrypted(deployment) // optional
    // hash: is the deployment predictable hash. the node must use the same method to calculate the challenge (bytes) to compute this same hash.
    //used for validating the deployment from node side.
    hash: hash(deployment)
    // ips: number of ips that need to be reserved by the contract and used by the deployment
    ips: 0
}
```
- After contract creation is successful, the user sends the **FULL** deployment to the node plus the contract ID. In this case, the contract ID is assumed to be the deployment unique ID.
- The node then will get the contract object from the chain
- Validation of the node address that it matches _this_ node.
- Validation of the twin (user) signature
- The node will recompute the hash and compare it against the contract hash as a sort of validation.
- If validation is successful, deployment is applied.
- Node start report capacity consumption to the blockchain, the contract then can bill the user.
