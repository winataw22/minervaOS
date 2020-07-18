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

## ReservationInfo
```go
type ReservationInfo struct {
	WorkloadId int64  
	NodeId     string 
	PoolId     int64  

	// Referene to an old reservation, used in conversion
	Reference string 

	Description             string         
	SigningRequestProvision SigningRequest 
	SigningRequestDelete    SigningRequest 

	ID                  schema.ID          
	Json                string             
	CustomerTid         int64              
	CustomerSignature   string             
	NextAction          NextActionEnum     
	SignaturesProvision []SigningSignature 
	SignatureFarmer     SigningSignature   
	SignaturesDelete    []SigningSignature 
	Epoch               schema.Date        
	Metadata            string             
	Result              Result             
	WorkloadType        WorkloadTypeEnum   
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
	ReservationInfo 

	Flist             string              
	HubUrl            string              
	Environment       map[string]string   
	SecretEnvironment map[string]string   
	Entrypoint        string              
	Interactive       bool                
	Volumes           []ContainerMount    
	NetworkConnection []NetworkConnection 
	StatsAggregator   []StatsAggregator   
	Logs              []Logs              
	Capacity          ContainerCapacity   
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
	ReservationInfo 

	Size            int64             
	ClusterSecret   string            
	NetworkId       string            
	Ipaddress       net.IP            
	MasterIps       []net.IP          
	SshKeys         []string          
	StatsAggregator []StatsAggregator 
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
type NetworkResource struct {
	ReservationInfo 

	Name                         string            
	NetworkIprange               schema.IPRange    
	WireguardPrivateKeyEncrypted string            
	WireguardPublicKey           string            
	WireguardListenPort          int64             
	Iprange                      schema.IPRange    
	Peers                        []WireguardPeer   
	StatsAggregator              []StatsAggregator 
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
	ReservationInfo 

	Size int64          
	Type VolumeTypeEnum 
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
	ReservationInfo 

	Size            int64             
	Mode            ZDBModeEnum       
	Password        string            
	DiskType        DiskTypeEnum      
	Public          bool              
	StatsAggregator []StatsAggregator 
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
type VolumeTypeEnum uint

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

```go
type GatewayProxy struct {
	ReservationInfo 

	Domain  string 
	Addr    string 
	Port    uint32 
	PortTLS uint32 
}
```

```go
type GatewayReverseProxy struct {
	ReservationInfo 

	Domain string 
	Secret string 
}
```

```go
type GatewaySubdomain struct {
	ReservationInfo 

	Domain string   
	IPs    []string 
}
```

```go
type GatewayDelegate struct {
	ReservationInfo 

	Domain string 
}
```

```go
type Gateway4To6 struct {
	ReservationInfo 

	PublicKey string 
}
```
