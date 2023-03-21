# 1. Dynamic Cache

Date: 2023-03-19

## Status

Accepted

## Context

Nodes used to have a fixed 100G of ssd storage reserved as cache for ZOS and all
downloaded files. This was a waste of space sense the normal used space out of this cache
is always around few hundred of MBs.

## Decision

Dynamic cache allocation. The system will start by reserving 5 GB of ssd for cache, then
dynamically increase of more space is needed. The system will also shrink the cache space
back if it's not used anymore.

This way more space will available for users workloads

## Consequences
