# IPerf

### Overview

The `iperf` package is designed to facilitate network performance testing using the `iperf3` tool. with both UDP and TCP over IPv4 and IPv6.

### Configuration

- Name: `iperf`
- Schedule: 4 times a day
- Jitter: 20 min

### Details

- The package using the iperf binary to examine network performance under different conditions.
- It randomly fetch PublicConfig data for randomly public nodes on the chain + all public node from free farm. These nodes serves as the targets for the iperf tests.
- For each node, it run the test with 4 times. through (UDP/TCP) using both node IPs (v4/v6)
- result will be a slice of all public node report (4 for each) each one will include:
  ```
    UploadSpeed: Upload speed (in bits per second).
    DownloadSpeed: Download speed (in bits per second).
    NodeID: ID of the node where the test was conducted.
    NodeIpv4: IPv4 address of the node.
    TestType: Type of the test (TCP or UDP).
    Error: Any error encountered during the test.
    CpuReport: CPU utilization report (in percentage).
  ```

### Result sample

```json
{
  "description": "Test public nodes network performance with both UDP and TCP over IPv4 and IPv6",
  "name": "iperf",
  "result": [
    {
      "cpu_report": {
        "host_system": 2.4433388913571044,
        "host_total": 3.542919199613454,
        "host_user": 1.0996094859359695,
        "remote_system": 0.24430594945859846,
        "remote_total": 0.3854457128784448,
        "remote_user": 0.14115962407747246
      },
      "download_speed": 1041274.4792242317,
      "error": "",
      "node_id": 124,
      "node_ip": "88.99.30.200",
      "test_type": "tcp",
      "upload_speed": 1048549.3668460822
    },
    {
      "cpu_report": {
        "host_system": 0,
        "host_total": 0,
        "host_user": 0,
        "remote_system": 0,
        "remote_total": 0,
        "remote_user": 0
      },
      "download_speed": 0,
      "error": "unable to connect to server - server may have stopped running or use a different port, firewall issue, etc.: Network unreachable",
      "node_id": 124,
      "node_ip": "2a01:4f8:10a:710::2",
      "test_type": "tcp",
      "upload_speed": 0
    }
  ],
  "timestamp": 1700507035
}
```
