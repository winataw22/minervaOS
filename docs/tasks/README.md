# Performance Monitor Package

### Overview

The `perf` package is a performance monitor in `zos` nodes. it schedules tasks, cache their results and allows retrieval of these results through `RMB` calls.

### Flow

1. The `perf` monitor is started by the `noded` service in zos.
2. Tasks are registered with a schedule in the new monitor.
3. A bus handler is opened to allow result retrieval.

### Node Initialization check

To ensure that the node always has a test result available, a check is performed on node startup for all the registered tasks, if a task doesn't have any stored result, it will run immediately without waiting for the next scheduled time.

### Scheduling

- Tasks are scheduled using a 6 fields cron format. this format provides flexibility to define time, allowing running tasks periodically or at specific time.

- Each task have a jitter which is the max number of seconds the task could sleep before it runs, this happens to prevent all tests ending up running at exactly the same time. so for example if a task scheduled to run at `06:00` and it is jitter is `10` it is expected to run any when between `06:00` to `06:10`

### RMB commands

- `zos.perf.get`:

  - Payload: a payload type that contains the name of the test

    ```go
    type Payload struct {
      Name string
    }
    ```

    Possible values:

    - `"public-ip-validation"`
    - `"cpu-benchmark"`
    - `"healthcheck"`
    - `"iperf"`

  - Return: a single task result.

  - Possible Error: `ErrResultNotFound` if no result is stored for the given task.

- `zos.perf.get_all`:

  - Return: all stored results

The rmb direct client can be used to call these commands. check the [example](https://github.com/threefoldtech/tfgrid-sdk-go/blob/development/rmb-sdk-go/examples/rpc_client/main.go)

### Caching

Results are stored in a Redis server running on the node.

The key in redis is the name of the task prefixed with the word `perf`.
The value is an instance of `TaskResult` struct contains:

- Name of the task
- Timestamp when the task was run
- A brief description about what the task do
- The actual returned result from the task

Notes:

- Storing results by a key ensures each new result overrides the old one, so there is always a single result for each task.
- Storing results prefixed with `perf` eases retrieving all the results stored by this module.

### Registered tests

- [Public IP validation](./publicips.md)
- [CPU benchmark](./cpubench.md)
- [Health Check](./healthcheck.md)
- [IPerf](./iperf.md)
