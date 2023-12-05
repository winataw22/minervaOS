# 1. Skipping Uptime Reports

Date: 2023-11-24

## Status

Accepted

## Context

Skip uptime reports for unhealthy nodes.

## Decision

Nodes will not be sending uptime reports if the node is in an unusable state to reduce minting for unhealthy nodes. The decision for a node to be unhealthy is based on multiple checks on the node modules and the capacity reported by the node.

## Consequences

Farmers will need to make sure the nodes are in a healthy state and the capacity reported are valid in order for the minting to go through.
