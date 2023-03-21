# 1. No overhead for VM memory

Date: 2022-11-24

## Status

Accepted

## Context

ZOS was adding an overhead of 5% for each VM it creates. So on creation of a VM the system will check if the request memory + min(5% of vm.mem, 1G) will fit inside the node available memory.
This was wrong because the calculated used memory on the system (comes from active deployments) does not take into account this overhead nor the system. Hence it was impossible for clients to request all available memory on the node.

Note that the node already reserved 10% of the machine memory for the system, this will account for the workload overhead (cloud-hypervisor memory + helper processes) anyway.

## Decision

- Drop the overhead assumed for a VM workload

## Consequences

A user can create a VM that allocate all computed free memory on the node.
