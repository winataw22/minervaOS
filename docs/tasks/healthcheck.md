# HealthCheck

## Overview

Health check task executes some checks over ZOS components to determine if the node is in a usable state or not and set flags for the Power Daemon to stop uptime reports if the node is unusable.

## Configuration

- Name: `healthcheck`
- Schedule: Every 15 mins.

## Details

- `cacheCheck`:
   Check if the node cache disk is usable or not by trying to write some data to it. If it failed, it set the Readonly flag.
- `networkCheck`:
   Check if the node can connect to the grid services chain, relay, hub, graphql, ...

## Result Sample

```json
{
   "description": "health check task runs multiple checks to ensure the node is in a usable state and set flags for the power daemon to stop reporting uptime if it is not usable",
   "name": "healthcheck",
   "result": {
      "cache": [
         "failed to write to cache: open /var/cache/healthcheck: operation not permitted"
      ]
   },
   "timestamp": 1701599580
}
```
