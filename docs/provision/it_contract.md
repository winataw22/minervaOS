# contract for IT

## Reservation

The reservation model consists of 2 parts: the reservation data, and the
reservation state.  
The data is is defined entirely when the reservation is created, and is immutable after this.  
The state is mutable, and after the reservation has been created, this will be updated continuously by both the customer threebot and the farmer threebot (and possibly other threebots who will sign to trigger actions).

### Reservation data

The reservation data is composed of the individual workloads to be deployed, information
about who can sign for the actual deployment and deletion of the workloads, and expiry
times for the reservation. All fields in the data object are immutable after the
reservation is created (i.e. the customer signs the data). Modifications afterwards
will cause the customer signature to become invalid.

### Reservation state

The reservation state is updated throughout the lifetime of the reservation. It also
contains the signatures needed to have the farmer threebot take action. In order for
the farmer threebot to start provisioning the workloads, or delete the workloads,
the `signing_request_provision` and `signing_request_delete`, respectively, need to be filled with
valid signatures.  

A valid signature is a signature for the reservation data, with a private key owned by one of the threebots listed in the reservation data (in the `signatures_provision` and `signatures_delete` fields).  
These fields also define the minimum amount of signatures required.  
For example, a signature request for provisioning might list 3 threebot ids which can sign, but only specify a `quorum_min` of 2. As such, only 2 out of the 3 listed threebot ids would need to sign before the node is allowed to deploy the workloads.

#### signature validity

A signature is created by signing a piece of data using a private key. Afterwards,
the corresponding public key can be used to check if the signature is valid. A [signature field](#signingsignature) is valid if it meets the following conditions:

- It contains at least the minimum amount of signatures required, as defined in
the `quorum_min` field of the corresponding [signing request field](#signingrequest), 1 if there is no such corresponding signing request.
- All signatures are valid with a public key owned by a referenced threebot (referenced in the aforementioned accompanying [signing request field](#signingrequest) or possibly other field).

##### signature algorithm

- signature algorithm: [ed25519](https://ed25519.cr.yp.to/)
- public key size: 32 bytes
- private key size: 32 bytes
- signature size: 64 bytes

### Data model

The following is an overview of the types used, their fields, and what these fields are used for.  You can find a full definition of the types in [provisiond.md](provision.md)

#### [Reservation](provisiond.md#reservation)

The [reservation object](provisiond.md#reservation) is the high level object for dealing with a reservation on the threefold grid. It is composed of the [reservation.data](#reservationdata) object, which holds all data for the workloads covered by this reservation (and is immutable after it is created), and additional info for the reservation state.

- `DataReservation`: As explained, the data field holds the [reservation.data](#reservationdata) object, which contains all low level workloads to be provisioned. After the reservation is created, this field is immutable.
- json: TODO: this field needs to away ~~A representation of the `data` object, in json form. It is directly derived from
the `data` field. As such, it is also immutable. This is the actual input for the signature algorithm.~~

Next to the reservation data, there is also a reservation state. These fields describe the current state of the reservation, as well as the signatures provided by authorized threebots to advance the state of the reservation.

- `NextAction`: This field describes what action should be performed next. Given the [enum values](provisiond.mdNextActionEnum), we can roughly describe a reservation life cycle as follows:  

  - user create the reservation, initial status is `created`
  - user sends the reservation to the explorer, status goes from `create` to `sign`
  - user sign the reservation, status goes from `sign` to `pay`
  - as a result of registering the reservation on the explorer, the user got a list of transaction to do in other to pay the farmer involved in to the reservation. Once the user has actually executed the transactions, the explorer checks the token have actually arrived, the status goes from `pay` to `deploy`. Check the [payment documentation](reservation_payment.md) for more detail information on how to pay for a reservation.
  - when a reservation is has a state `deploy`, the node can now pick it up and provision the workloads.  

  From here there are 2 possibility:  

  - the reservation expires, it's state goes from `deploy` to `delete`.  
  - Or the user decide to delete the reservation before it expires. It validate the condition defined in `SignaturesDelete` field. the state does from `deploy` to `delete`
  - when a reservation is has a state `delete`. The node decommission the workloads, and reports the workloads to be deleted, and once all workloads have been marked as deleted, the reservation state goes from `delete` to `deleted` and the reservation life cycle is ended.

- `SignaturesProvision`: A list of `signatures` needed to start the provisioning (deploy) step.  
i.e. after enough valid signatures are provided here, the nodes can start to deploy the workloads defined. Validity of signatures and  amount of valid signatures required is defined by the `SigningRequestProvision` field in the [data]((provisiond.md#reservation-data) object.

~~- `SignaturesFarmer`: the [signatures](#signingsignature) of the farmer threebots, which declares that the farmer agrees to provision the workloads as defined by the reservation once there is consensus about the provisioning (see previous field). Every farmer who deploys a workload will need to sign this. To find out which threebots need to sign, you can iterate over the workloads defined in the [reservation data](#reservationdata), and collect a set of unique farmer id's from them.~~

- `SignaturesDelete`: Much like `SignaturesProvision`, however it is used when a currently deployed workload needs to be deleted (before it expires). It is tied to the `SignaturesDelete` field in the `data` object.
- `epoch`: The date of the last modification
- `results`: A list of [reservation results](#reservationresult). Every workload which is defined in the reservation
will return a result describing the status. This allows fine grained error handling for individual
workloads.

#### [Reservation.Data](provisiond.md#reservation-data)

The reservation data contains all required info for the workloads to be deployed,
as well as info about who can sign to start the provisioning and deletion, and the expiry
dates for the reservation. As the JSON representation of the data is signed by the customer
after creating the reservation, all of these fields are immutable after being created.

- description: Description of the reservation/workloads.
- containers: Container workloads to be provisioned, see https://github.com/threefoldtech/zosv2/blob/master/docs/provision/provision.md
- volumes: Volume workloads to be provisioned, see https://github.com/threefoldtech/zosv2/blob/master/docs/provision/provision.md
- zdbs: ZDB workloads to be provisioned, see https://github.com/threefoldtech/zosv2/blob/master/docs/provision/provision.md
- networks: Network workloads to be provisioned, see https://github.com/threefoldtech/zosv2/blob/master/docs/provision/provision.md
- expiration_provisioning: The expiry time of the provisioning step. If the provisioning signatures
in `signing_request_provision` are not collected before this time, the reservation is
considered invalid and a new one must be created.
- expiration_reservation: The expiry time of the reservation, i.e. the provisioned workloads.
The farmer(s) agree(s) to keep the provisioned workloads available until at least this
time.
- signing_request_provision: The list of threebots which can sign for the provisioning to happen,
and the minimum amount of signatures required to do so, as described in [signing request](#siginingrequest).
- signing_request_delete: The list of threebots which can sign for the early deletion of the workloads
to happen, and the minimum amount of signatures required to do so, as described in [signing request](#signingrequest).

#### [SigningRequest](provisiond.md#SigningRequest)

A signing request defines who (which threebots) can sign for a particular action,
and the minimum amount of required signatures. The minimum amount of people needed
can be anything between 1 and the number of signers.

- `Signers`: A list of threebot ids who can sign. To verify the signature, the public
key of the threebot can be loaded, and then used to verify the signature.
- `QuorumMin`: The minimum amount of requested signatures. At least this amount of threebots need to sign before the signature request is considered fulfilled.

As an example of how this might be applied in practice, consider the following
signing request:

- `Signers`: [threebot_a, threebot_b, threebot_c]
- `QuorumMin`: 1

This means that any of the 3 listed threebots can sign the data, and the request is fulfilled as soon as anyone signs. For instance, a workload for testing is used by 3 developers, and any of those can choose to have the workload deployed or deleted.  
If however another person signs (perhaps a 4th developer who is new in the company), the signature will not be valid, as he is not listed in the `signers` field, and therefore he is not able to deploy the workload.

Note that `quorum_min` is a _minimum_ and as such, it is possible, and legal, for more than 1 of the listed persons to sign.  
I.e. if both `threebot_a` and `threebot_b` sign, the request is still fulfilled.

#### [SigningSignature](provisiond.me#SigningSignature)

A signature has the actual `signature` bytes, as well as the id of the threebot which signed. The threebot id is used to verify that this threebot is actually allowed to sign, and to fetch its public key to verify the signature. Additionally, the time of signing is also recorded.

- `tid`: Id of the threebot which signed.
- `signature`: The actual signature in binary form
- `epoch`: Time of signing

#### [Reservation.Result](provisiond.md#Result)

A result is used by a 0-OS node to add a response to a reservation. This result can inform users if an error occurred, or more commonly, it can relay back vital information such as the IP address of a container after it is started.  
The result object has a `WorkloadId` field, which is used to map the result to the actual workload. With the workload request, the `NodeId` can be inspected, to get the nodes public key. The key can then be used to verify the signature of the data, proving that it is indeed this node which created the reply, and that the `DataJSON` (TODO:remove this field) has not been tampered with after it was created.

- `Category`: The type of workload for which the reply is.
- `WorkloadId`: The id of the workload for which the reply is. This will be the same as one of the `workload_id`s in the [reservation data](#reservationdata).
- `DataJson`: The full data as a json object.
- `Signature`: The bytes of the signature. The signature is created by the node which creates the reply. The data signed is the `data_json` field. This proves authenticity of the reply as well as integrity of the response data.
- `State`: Did the workload deploy ok ("ok") or not ("error").
- `Message`: Content of the message sent by the node.
- `Epoch`: Time at which the result has been created.
- `NodeId`: the node ID of the node that deployed the workload

## Reservation flow Diagram

![process](../../assets/grid_provisioning2.png)

## Actors

- **User**: digital avatar of a user. This is the one buying capacity on the grid.
- **Farmer**: digital avatar of a farmer. It owns some node in the grid and is responsible to set the price of its node capacity.
- **TF Explorer**: Public directory listing all the nodes/farmers in the grid
- **Blockchain**: money blockchain, money transaction are executed on this chain using ThreefoldToken (TFT).
- **Node**: Hardware running 0-OS and responsible to provide capacity to the TFGrid.
