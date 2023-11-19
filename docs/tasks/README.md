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

Tasks are scheduled using a 6 fields cron format. this format provides flexibility to define time, allowing running tasks periodically or at specific time.

### RMB commands

- `zos.perf.get`:
  Payload: string representing the task name.
  Return: a single task result.
  Possible Error: `ErrResultNotFound` if no result is stored for the given task.

- `zos.perf.get_all`:
  Return: all stored results

### Caching

Results are stored in a Redis server running on the node.

The key in redis is the name of the task prefixed with the word `perf`.
The value is an instance of `TaskResult` struct contains:

- Name of the task
- Timestamp when the task was run
- The actual returned result from the task

Notes:

- Storing results by a key ensures each new result overrides the old one, so there is always a single result for each task.
- Storing results prefixed with `perf` eases retrieving all the results stored by this module.

### Registered tests
- [Public IP validation](./publicips.md)
- [CPU benchmark](./cpubench.md)
- [IPerf](./iperf.md)
- To add new task, [check](../../pkg/perf/README.md) 