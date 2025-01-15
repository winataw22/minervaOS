# RMB Client Documentation

## Package Overview

The `client` package provides a simple RMB interface to work with nodes.

---

## Requirements

1. **Message Bus (msgbusd):** A `msgbusd` instance must be running on the node. This client uses RMB to send messages and receive responses.
2. **ED25519 Key Pair:** A valid ed25519 key pair is required. This key is used to sign deployments and **must** match the key configured for the local twin on substrate.

---

## Simple Deployment Example

### Step 1: Create an RMB Client

```go
cl, err := rmb.Default()
if err != nil {
    panic(err)
}
```

### Step 2: Create a Node Client

```go
node := client.NewNodeClient(NodeTwinID, cl)
```

### Step 3: Define Your Deployment Object

```go
dl := gridtypes.Deployment{
    Version: Version,
    TwinID:  Twin, // LocalTwin,
    // This contract ID must match the one on substrate
    Workloads: []gridtypes.Workload{
        network(), // Network workload definition
        zmount(),  // Zmount workload definition
        publicip(), // Public IP definition
        zmachine(), // Zmachine definition
    },
    SignatureRequirement: gridtypes.SignatureRequirement{
        WeightRequired: 1,
        Requests: []gridtypes.SignatureRequest{
            {
                TwinID: Twin,
                Weight: 1,
            },
        },
    },
}
```

### Step 4: Compute Deployment Hash

```go
hash, err := dl.ChallengeHash()
if err != nil {
    panic("failed to create hash")
}
fmt.Printf("Hash: %x\n", hash)
```

### Step 5: Deploy the Contract

```go
dl.ContractID = 11 // Contract ID from substrate
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err = node.DeploymentDeploy(ctx, dl)
if err != nil {
    panic(err)
}
```

## Node Client Methods

### Deployment Management

#### Deployment Deploy

Sends the deployment to the node for processing.

```go
func (n *NodeClient) DeploymentDeploy(ctx context.Context, dl gridtypes.Deployment) error
```

#### Deployment Update

Updates a given deployment.

> Deployment must be a valid update for a deployment that has been already created via DeploymentDeploy

```go
func (n *NodeClient) DeploymentUpdate(ctx context.Context, dl gridtypes.Deployment) error
```

#### Deployment Get

Gets a deployment via contract ID

```go
func (n *NodeClient) DeploymentGet(ctx context.Context, contractID uint64) (gridtypes.Deployment, error)
```

#### Deployment List

List all deployments on the node for a twin.

```go
func (n *NodeClient) DeploymentList(ctx context.Context) ([]gridtypes.Deployment, error)
```

#### Deployment Changes

Get changes to a deployment by contract ID.

```go
func (n *NodeClient) DeploymentChanges(ctx context.Context, contractID uint64) ([]gridtypes.Workload, error)
```

#### Deployment Delete

Delete a deployment.

```go
func (n *NodeClient) DeploymentDelete(ctx context.Context, contractID uint64) error
```

---

### Node Statistics

#### Get Counters

Gets node statistics, including total and available cpu, memory, storage, etc...

```go
func (n *NodeClient) Counters(ctx context.Context) (Counters, error)
```

#### Pools

Returns the statistics of separate pools

```go
func (n *NodeClient) Pools(ctx context.Context) ([]pkg.PoolMetrics, error)
```

---

### GPU Management

#### List GPUs

Gets a list of GPUs on the node.

```go
func (n *NodeClient) GPUs(ctx context.Context) ([]GPU, error)
```

---

### Networking

#### List WireGuard Ports

List return a list of all "taken" ports on the node.

```go
func (n *NodeClient) NetworkListWGPorts(ctx context.Context) ([]uint16, error)
```

#### Check Public IPv6 Availability

Check if the node has a public IP of version 6 address.

```go
func (n *NodeClient) HasPublicIPv6(ctx context.Context) (bool, error)
```

#### List Interfaces

Retrieve all interfaces on the node.

```go
func (n *NodeClient) NetworkListInterfaces(ctx context.Context) (map[string][]net.IP, error)
```

#### List Public IPs

List taken public IPs on the node

```go
func (n *NodeClient) NetworkListPublicIPs(ctx context.Context) ([]string, error)
```

#### List Private IPs

Retrieve all private IPs reserved for a network.

```go
func (n *NodeClient) NetworkListPrivateIPs(ctx context.Context, networkName string) ([]string, error)
```

#### Get Public Network Configuration

Retuns the current public network configuration for the node.

```go
func (n *NodeClient) NetworkGetPublicConfig(ctx context.Context) (pkg.PublicConfig, error)
```

---

### System Information

#### Get System Version

Returns the system version.

```go
func (n *NodeClient) SystemVersion(ctx context.Context) (Version, error)
```

#### Get Node Features

Gets features of the node (This can be used to indicate if the node is of version 3 or 4).

```go
func (n *NodeClient) SystemGetNodeFeatures(ctx context.Context) ([]pkg.NodeFeature, error)
```

#### Get System DMI

Returns DMI information for the node.

```go
func (n *NodeClient) SystemDMI(ctx context.Context) (dmi.DMI, error)
```

#### Get Hypervisor Information

Gets the name of the hypervisor used on the node

```go
func (n *NodeClient) SystemHypervisor(ctx context.Context) (string, error)
```

#### Run Diagnostics

Runs diagnostics on the system.

```go
func (n *NodeClient) SystemDiagnostics(ctx context.Context) (diagnostics.Diagnostics, error)
```

---

### Calls requires admin privileges

#### List All Physical Interfaces

List all physical devices on a node

```go
func (n *NodeClient) NetworkListAllInterfaces(ctx context.Context) (map[string]Interface, error)
```

#### Set Public Exit Device

Set which physical interface to use as the exit device.

```go
func (n *NodeClient) NetworkSetPublicExitDevice(ctx context.Context, iface string) error
```

#### Get Public Exit Device

Get the current dual NIC setup of the node.

```go
func (n *NodeClient) NetworkGetPublicExitDevice(ctx context.Context) (ExitDevice, error)
```

---

## Structs and Types

### NodeClient

Represents the node client.

```go
type NodeClient struct {
    nodeTwin uint32
    bus      rmb.Client
}
```

### Version

Represents system version information.

```go
type Version struct {
    ZOS   string `json:"zos"`
    ZInit string `json:"zinit"`
}
```

### Interface

Represents network interface information.

```go
type Interface struct {
    IPs []string `json:"ips"`
    Mac string   `json:"mac"`
}
```

### ExitDevice

Represents exit device configuration.

```go
type ExitDevice struct {
    IsSingle       bool   `json:"is_single"`
    IsDual         bool   `json:"is_dual"`
    AsDualInterface string `json:"dual_interface"`
}
```

### Counters

Represents node statistics.

```go
type Counters struct {
    Total  gridtypes.Capacity `json:"total"`
    Used   gridtypes.Capacity `json:"used"`
    System gridtypes.Capacity `json:"system"`
    Users  UsersCounters      `json:"users"`
}
```

### UsersCounters

Represents deployment and workload statistics.

```go
type UsersCounters struct {
    Deployments int `json:"deployments"`
    Workloads   int `json:"workloads"`
}
```

### GPU

Represents GPU information.

```go
type GPU struct {
    ID       string `json:"id"`
    Vendor   string `json:"vendor"`
    Device   string `json:"device"`
    Contract uint64 `json:"contract"`
}
```
