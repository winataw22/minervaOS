# 0-OS v2 Provisioning schemas

## Reservation
```go
type Reservation struct {
	ID                  int64
	Json                string
	DataReservation     Data
	CustomerTid         int64
	CustomerSignature   string
	NextAction          NextActionEnum
	SignaturesProvision []SigningSignature
	SignaturesFarmer    []SigningSignature
	SignaturesDelete    []SigningSignature
	Epoch               time.Time
	Results             []Result
}
```

## Reservation Data
```go
type Data struct {
	Description             string
	SigningRequestProvision SigningRequest
	SigningRequestDelete    SigningRequest
	Containers              []Container
	Volumes                 []Volume
	Zdbs                    []Zdb
	Networks                []Network
	Kubernetes              []K8S
	ExpirationProvisioning  time.Time
	ExpirationReservation   time.Time
}
```

### SigningRequest
```go
type SigningRequest struct {
	Signers   []int64
	QuorumMin int64
}
```

### SigningSignature
```go
type SigningSignature struct {
	Tid       int64
	Signature string
	Epoch     time.Time
}
```

```go
type Container struct {
	WorkloadId        int64
	NodeId            string
	Flist             string
	HubUrl            string
	Environment       map[string]interface{}
	SecretEnvironment map[string]interface{}
	Entrypoint        string
	Interactive       bool
	Volumes           []ContainerMount
	NetworkConnection []NetworkConnection
	StatsAggregator   []Statsaggregator
	Logs              []Logs
	FarmerTid         int64
}
```

```go
type Logs struct {
	Type string
	Data LogsRedis
}
```

```go
type LogsRedis struct {
	Stdout string
	Stderr string
}
```

```go
type ContainerMount struct {
	VolumeId   string
	Mountpoint string
}
```

```go
type NetworkConnection struct {
	NetworkId string
	Ipaddress net.IP
}
```

```go
type K8S struct {
	WorkloadId      int64
	NodeId          string
	Size            int64
	NetworkId       string
	Ipaddress       net.IP
	ClusterSecret   string
	MasterIps       []net.IP
	SshKeys         []string
	StatsAggregator []Statsaggregator
	FarmerTid       int64
}
```

```go
type Network struct {
	Name             string
	WorkloadId       int64
	Iprange          schema.IPRange
	StatsAggregator  []Statsaggregator
	NetworkResources []NetworkNetResource
	FarmerTid        int64
}
```

```go
type NetworkNetResource struct {
	NodeId                       string
	WireguardPrivateKeyEncrypted string
	WireguardPublicKey           string
	WireguardListenPort          int64
	Iprange                      schema.IPRange
	Peers                        []WireguardPeer
}
```

```go
type WireguardPeer struct {
	PublicKey      string
	AllowedIprange []schema.IPRange
	Endpoint       string
	Iprange        schema.IPRange
}
```

### Result
```go
type Result struct {
	Category   ResultCategoryEnum
	WorkloadId string
	DataJson   json.RawMessage
	Signature  string
	State      ResultStateEnum
	Message    string
	Epoch      time.Time
	NodeId     string
}
```

```go
type Statsaggregator struct {
	Addr   string
	Port   int64
	Secret string
}
```

```go
type Volume struct {
	WorkloadId      int64
	NodeId          string
	Size            int64
	Type            VolumeTypeEnum
	StatsAggregator []Statsaggregator
	FarmerTid       int64
}
```

```go
type Workload struct {
	WorkloadId string
	User       string
	Type       WorkloadTypeEnum
	Content    interface{}
	Created    time.Time
	Duration   int64
	Signature  string
	ToDelete   bool
}
```

```go
type Zdb struct {
	WorkloadId      int64
	NodeId          string
	Size            int64
	Mode            ZdbModeEnum
	Password        string
	DiskType        ZdbDiskTypeEnum
	Public          bool
	StatsAggregator []Statsaggregator
	FarmerTid       int64
}
```

```go
type ZdbDiskTypeEnum uint8

const (
	ZdbDiskTypeHdd ZdbDiskTypeEnum = iota
	ZdbDiskTypeSsd
)
```

## NextActionEnum
```go
type NextActionEnum uint8

const (
	NextActionCreate NextActionEnum = iota
	NextActionSign
	NextActionPay
	NextActionDeploy
	NextActionDelete
	NextActionInvalid
	NextActionDeleted
)
```

```go
type ResultCategoryEnum uint8

const (
	ResultCategoryZdb ResultCategoryEnum = iota
	ResultCategoryContainer
	ResultCategoryNetwork
	ResultCategoryVolume
)
```

```go
type ResultStateEnum uint8

const (
	ResultStateError ResultStateEnum = iota
	ResultStateOk
	ResultStateDeleted
)
```

```go
type VolumeTypeEnum uint8

const (
	VolumeTypeHDD VolumeTypeEnum = iota
	VolumeTypeSSD
)
```

```go
type WorkloadTypeEnum uint8

const (
	WorkloadTypeZdb WorkloadTypeEnum = iota
	WorkloadTypeContainer
	WorkloadTypeVolume
	WorkloadTypeNetwork
	WorkloadTypeKubernetes
)
```

```go
type ZdbModeEnum uint8

const (
	ZdbModeSeq ZdbModeEnum = iota
	ZdbModeUser
)
```