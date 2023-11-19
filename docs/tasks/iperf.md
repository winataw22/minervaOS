# IPerf

### Overview

The `iperf` package is designed to facilitate network performance testing using the `iperf3` tool. with both UDP and TCP over IPv4 and IPv6.

### Configuration

- Name: iperf
- Schedule: 4 times a day

### Details

- The package using the iperf binary to examine network performance under different conditions.
- It randomly fetch PublicConfig data for randomly public nodes on the chain + all public node from free farm. These nodes serves as the targets for the iperf tests.
- For each node, it run the test with 4 times. through (UDP/TCP) using both node IPs (v4/v6)
- result will be a slice of all public node report (4 for each) each one will include:
  ```
    UploadSpeed: Upload speed in bits per second.
    DownloadSpeed: Download speed in bits per second.
    NodeID: ID of the node where the test was conducted.
    NodeIpv4: IPv4 address of the node.
    TestType: Type of the test (TCP or UDP).
    Error: Any error encountered during the test.
    CpuReport: CPU utilization report.
  ```
