# CPUBenchmark

### Overview

The `CPUBenchmark` task is designed to measure the performance of the CPU. it utilizes the [cpu-benchmark-simple](https://github.com/threefoldtech/cpu-benchmark-simple) tool and includes a zos stub to gather the number of workloads running on the node.

### Configuration

- Name: `cpu-benchmark`
- Schedule: 4 times a day

### Details

- The benchmark simply runs a `CRC64` computation task, calculates the time spent in the computation and reports it in `seconds`.
- The computation is performed in both single-threaded and multi-threaded scenarios.
- Lower time = better performance: for a single threaded benchmark, a lower execution time indicates better performance.

### Result sample
```json
{
  "description": "Measures the performance of the node CPU by reporting the timespent of computing a task in seconds.",
  "name": "cpu-benchmark",
  "result": {
    "multi": 1.105,
    "single": 1.135,
    "threads": 1,
    "workloads": 0
  },
  "timestamp": 1700504403
}
```