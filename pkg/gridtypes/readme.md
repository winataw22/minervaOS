# Grid Types
Those are types that are used to communicate with a zos node.

## Workload
This is the main envelope that hold all reservation information

```go
type Workload struct {
	//Version (optional) is version of reservation object
	Version int `json:"version"`
	// ID of the reservation (filled by the node)
	ID ID `json:"id"`
	// User (required) of the user requesting the reservation
	User ID `json:"user_id"`
	// Type (required) of the reservation (container, zdb, vm, etc...)
	Type WorkloadType `json:"type"`
	// Data (required) is the reservation type arguments. It's different per Type
	Data json.RawMessage `json:"data"`
	// Date of creation (filled by the node)
	Created time.Time `json:"created"`
	//ToDelete is set if the user/farmer asked the reservation to be deleted
	ToDelete bool `json:"to_delete"`
	// Metadata (optional) is custom user metadata
	Metadata string `json:"metadata"`
	//Description (optional)
	Description string `json:"description"`
	// User signature (required)
	Signature string `json:"signature"`
	// Result of reservation (filled by the node)
	Result Result `json:"result"`
}
```

The signature is filled up by computing a challenge message from the Workload data, then the signature is filled as
```
signature = hex(ed25591.sign(sk, challenge))
```
> please check the implementation in this package how the challenge is computed from the workload data.

## WorkloadType
## Data
For each workload type, the `Data` must be filled with proper parameters for this workload types.

### Zmount
check Zmount data [here](zos/zmount.go)

### ZDB
check zdb data [here](zos/zdb.go)

### Network
check network data [here](zos/network.go)

### IPV4
check ipv4 data [here](zos/ipv4.go)

### Zmachine
check zmachine data [here](zos/zmachine.go)

### Kubernetes
check k8s data [here](zos/kubernetes.go)
