# CPUBenchmark

### Overview

The `CPUBenchmark` task is designed to measure the performance of the CPU. it utilizes the [cpu-benchmark-simple](https://github.com/threefoldtech/cpu-benchmark-simple) tool and includes a zos stub to gather the number of workloads running on the node.

### Configuration

- Name: `cpu-benchmark`
- Schedule: 4 times a day
- Jitter: 0

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

### Result explanation

The best way to know what's a good or bad value is by testing and comparing different hardware.
Here are some example:

**1x Intel(R) Xeon(R) W-2145 CPU @ 3.70GHz** (Q3'2017)

```
Single thread score: 0.777
Multi threads score: 13.345 [16 threads]
```

**1x Intel(R) Pentium(R) CPU G4400 @ 3.30GHz** (Q3'2015)

```
Single thread score: 1.028
Multi threads score: 2.089 [2 threads]
```

**1x Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz** (Q2'2012)

```
Single thread score: 2.943
Multi threads score: 12.956 [4 threads]
```

**2x Intel(R) Xeon(R) CPU E5-2630 v3 @ 2.40GHz** (Q1'2012)

```
Single thread score: 1.298
Multi threads score: 44.090 [32 threads]
```

**2x Intel(R) Xeon(R) CPU L5640 @ 2.27GHz** (Q1'2010)

```
Single thread score: 2.504
Multi threads score: 72.452 [24 threads]
```

As you can see, the more recent the CPU is, the faster it is, but for a same launch period, you can see Xeon way better than regular/desktop CPU. You have to take in account the amount of threads and the time per threads.
