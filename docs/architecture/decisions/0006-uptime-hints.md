# 1. Dynamic Cache

Date: 2023-06-14

## Status

Accepted

## Context

Use tfchain uptimeV2 instead of old deprecated v1 uptime

## Decision

The uptime v2 also give an `timestamp hint` which is the time when the node sent the report as per the node itself. The chain
will then accept the report if and only if the hint is in an acceptable range of the chain local timestamp.

This is to prevent reports that have been buffered (delayed) on either sides of the client or the server from still getting processed
and causing inconsistency in the uptime report for that node.

Now nodes that are going to use v2 will get an error if the report is too old (or in the future)

## Consequences
