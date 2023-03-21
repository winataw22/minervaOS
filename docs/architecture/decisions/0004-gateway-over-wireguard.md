# 1. Gateway to user NR

Date: 2023-03-14

## Status

Accepted

## Context

Support gateway over user private wireguard. This means private workloads can be exposed (over the gateway) without using
yggdrasil.

Users can accomplish this by adding the gateway node to their network (can also act as an access-point) then while configuring
the gateway they need to set the network name.

## Decision

- User can provide a private ip to the target (backend) now for gateway workloads
- User has to include the gateway node into his network
- Gateway workload has to include network name where the ip lives
- Backward compatible with old configuration. backend still can be over yggdrail

## Consequences
