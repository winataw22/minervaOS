# Capacity

This document describes how ZOS does the following tasks:

- Reserved system resources
  - Memory
  - Storage
- Calculation of free usable capacity for user workloads

## System reserved capacity

ZOS always reserve some amount of the available physical resources to its own operation. The system tries to be as protective
as possible of it's critical services to make sure that the node is always reachable and usable even if it's under heavy load

ZOS make sure it reserves Memory and Storage (but not CPU) as per the following:

### Reserved Memory

ZOS reserve 10% of the available system memory for basic services AND operation overhead. The operation overhead can happen as a side effect of running user workloads. For example, a user network while in theory does not consume any memory, in matter of fact it also consume some memory (kernel buffers, etc...). Same for a VM. A user VM can be assigned say 5G but the process that running the VM can/will take few extra megabytes to operate.

This is why we decided to play on the safe side, and reserve 10% of total system memory to the system overhead, with a **MIN** reserved memory of 2GB

```python
reserved = min(total_in_gb * 0.1, 2G)
```

### Reserved Storage

While ZOS does not require installation, but it needs to download and store many things to operate correctly. This include the following:

- Node identity. Information about the node id and keys
- The system binaries, those what include all zos to join the grid and operate as expected
- Workload flists. Those are the flists of the user workloads. Those are downloaded on demand so they don't always exist.
- State information. Tracking information maintained by ZOS to track the state of workloads, owner-ship, and more.

This is why the system on first start allocates and reserve a part of the available SSD storage and is called `zos-cache`. Initially is `5G` (was 100G in older version) but because the `dynamic` nature of the cache we can't fix it at `5G`

The required space to be reserved by the system can dramatically change based on the amount of workloads running on the system. For example if many users are running many different VMs, the system will need to download (and cache) different VM images, hence requiring more cache.

This is why the system periodically checks the reserved storage and then dynamically expand or shrink to a more suitable value in increments of 5G. The expansion happens around the 20% of current cache size, and shrinking if went below 20%.

## User Capacity

All workloads requires some sort of a resource(s) to run and that is actually what the user hae to pay for. Any workload can consume resources in one of the following criteria:

- CU (compute unit in vCPU)
- MU (memory unit in bytes)
- NU (network unit in bytes)
- SU (ssd storage in bytes)
- HU (hdd storage in bytes)

A workloads, based on the type can consume one or more of those resource types. Some workloads will have a well known "size" on creation, others might be dynamic and won't be know until later.

For example, a disk workload SU consumption will be know ahead. Unlike the NU used by a network which will only be known after usage over a certain period of time.

A single deployment can has multiple workloads each requires a certain amount of one or more capacity types (listed above). ZOS then for each workloads type compute the amount of resources needed per workload, and then check if it can provide this amount of capacity.

> This means that a deployment that define 2 VMs can partially succeed to deploy one of the VMs but not the other one if the amount of resources it requested are higher than what the node can provide

### Memory

How the system decide if there are enough memory to run a certain workload that demands MU resources goes as follows:

- compute the "theoretically used" memory by all user workloads excluding `self`. This is basically the sum of all consumed MU units of all active workloads (as defined by their corresponding deployments, not as per actually used in the system).
- The theoretically used memory is topped with the system reserved memory.
- The the system checks actually used memory on the system this is done simply by doing `actual_used = memory.total - memory.available`
- The system now can simply `assume` an accurate used memory by doing `used = max(actual_used, theoretically_used)`
- Then `available = total - used`
- Then simply checks that `available` memory is enough to hold requested workload memory!

### Storage

Storage is much simpler to allocate than memory. It's completely left to the storage subsystem to find out if it can fit the requested storage on the available physical disks or not, if not possible the workloads is marked as error.

Storage tries to find the requested space based on type (SU or HU), then find the optimal way to fit that on the available disks, or spin up a new one if needed.
