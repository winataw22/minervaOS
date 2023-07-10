# 1. Dynamic Cache

Date: 2023-07-06

## Status

Accepted

## Context

Block SMTP traffic for private nodes

## Decision

Workloads that are hidden (has no public IPs) cannot send traffic over 25, 587, 465. This is to prevent reduce spam because
many farmers (specially home farmers) got warnings because of detected spam traffic

## Consequences

Workloads in private farms can not send or receive SMTP traffic. A user willing to establish smtp traffic need to either use
a public node or have public ip
