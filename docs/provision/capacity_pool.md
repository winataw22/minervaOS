# Capacity pool

Deploying a workload on the tf grid happens in 2 steps:

1. Reserve a capacity pool, this is your actual capacity reservation.
1. Deploy a workloads connected to this pool.

The capacity pool can be thought of as "an amount of capacity, ready to use some
nodes". Which nodes you can use a pool on is defined in the reservation for the pool.
The capacity is represented as `CUs` and `SUs`, `CU seconds` and `SU seconds` respectively.
As the name implies, each of these units gives you the ability to deploy a workload
worth 1 CU/SU for one second. If your workload requires, for example, 2 CU per second,
then your pool will be drained by 2 CU every second. The amount of time left in
a pool can thus be described as the minimum of the amount of CUs left divided by
the active CU and the amount of SUs left divided by the active SU. At any point
in time, it is possible to increase the amount of CUs or SUs in the pool by creating
a new capacity reservation referencing the old one. In doing so, a nearly empty
pool can be refilled, allowing associated workloads to continue operating. This
means that it is possible to have a workload running forever.

It is also possible to implement a system where you make recurring capacity
reservations and payments. For instance, a pool can be created which is large enough
to support a workload for 1 month. After this month, the pool can be refilled for
another month, etc. The time period can also be decreased should this be desired.

Finally, deploying new workloads with an existing pool is perfectly possible. Naturally,
this means that the pool will be empty sooner, and will need to be filled up quicker.
Conversely, if multiple workloads are associated with a pool and some of them are
stopped, the pool will last longer with the remaining workloads.

## Reserving a pool

Reserving a pool is done with a capacity reservation. This mimics the idea used
in the workload deployment via the smart contract for it. Specifically, the structure
to reserve a capacity pool looks like this:

```go
Reservation struct {
	JSON              string
	DataReservation   ReservationData
	CustomerTid       int64
	CustomerSignature string
}

ReservationData struct {
	PoolID     int64
	CUs        uint64
	SUs        uint64
	NodeIDs    []string
	Currencies []string
}
```

### Pool reservation fields

- **JSON**: json representation of the `ReservationData` object
- **DataReservation**: the actual data, signed and immutable. For more info
- **CustomerTid**: The threebot id of the customer creating this reservation,
this will also become the owner of the pool. Only the owner of this threebot
can then deploy workloads on the pool.
- **CustomerSignature**: The hex encoded signature of the JSON data.

- **PoolID**: the id of a pool to refill. If this is a reservation for a new pool,
this field can be omitted (or set to the default of 0)
- **CUs**: the amount of `CUs` to add to the pool
- **SUs**: the amount of `SUs` to add to the pool
- NodeIDs: the ids of the nodes in this pool. After the pool is created, workloads
can be created on any of these nodes tied to this pool. Note that it is currently
only possible to select nodes belonging to the same farm.
- **Currencies**: the currencies the client wants to pay in. If no currency is
accepted by the farmer (farmer is derived from the list of node IDs), the reservation
will fail.

## Using a pool

Once the capacity reservation has been created, payment information will be returned.
This information will return the id of the reservation in the system. If a new pool
is created, this id will also be the id of the pool. To deploy a workload with
your newly reserved capacity, you simpley need to set the `PoolID` field of the
workload reservation to this ID. If you top up a pool, the id will stay the same.
Note that a new id will be returned as well since the id is also used for the payment.

With the capacity reservation ID, you can pay for the reservation by adding a memo
`p-{id}` in your payment. For example, paying for capacity reservation 5 (which
could create a pool with id 5, or fill up an existing pool should you so choose),
the memo would need to be `p-5`.
