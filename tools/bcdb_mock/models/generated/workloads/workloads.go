package workloads

import (
	"encoding/json"
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

type Reservation struct {
	ID                  schema.ID          `bson:"_id" json:"id"`
	Json                string             `bson:"json" json:"json"`
	DataReservation     ReservationData    `bson:"data_reservation" json:"data_reservation"`
	CustomerTid         int64              `bson:"customer_tid" json:"customer_tid"`
	CustomerSignature   string             `bson:"customer_signature" json:"customer_signature"`
	NextAction          NextActionEnum     `bson:"next_action" json:"next_action"`
	SignaturesProvision []SigningSignature `bson:"signatures_provision" json:"signatures_provision"`
	SignaturesFarmer    []SigningSignature `bson:"signatures_farmer" json:"signatures_farmer"`
	SignaturesDelete    []SigningSignature `bson:"signatures_delete" json:"signatures_delete"`
	Epoch               schema.Date        `bson:"epoch" json:"epoch"`
	Results             []Result           `bson:"results" json:"results"`
}

func NewReservation() (Reservation, error) {
	const value = "{\"json\": \"\"}"
	var object Reservation
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type ReservationData struct {
	Description             string         `bson:"description" json:"description"`
	SigningRequestProvision SigningRequest `bson:"signing_request_provision" json:"signing_request_provision"`
	SigningRequestDelete    SigningRequest `bson:"signing_request_delete" json:"signing_request_delete"`
	Containers              []Container    `bson:"containers" json:"containers"`
	Volumes                 []Volume       `bson:"volumes" json:"volumes"`
	Zdbs                    []ZDB          `bson:"zdbs" json:"zdbs"`
	Networks                []Network      `bson:"networks" json:"networks"`
	Kubernetes              []K8S          `bson:"kubernetes" json:"kubernetes"`
	ExpirationProvisioning  schema.Date    `bson:"expiration_provisioning" json:"expiration_provisioning"`
	ExpirationReservation   schema.Date    `bson:"expiration_reservation" json:"expiration_reservation"`
}

func NewReservationData() (ReservationData, error) {
	const value = "{\"description\": \"\"}"
	var object ReservationData
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type SigningRequest struct {
	Signers   []int64 `bson:"signers" json:"signers"`
	QuorumMin int64   `bson:"quorum_min" json:"quorum_min"`
}

func NewSigningRequest() (SigningRequest, error) {
	const value = "{}"
	var object SigningRequest
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type SigningSignature struct {
	Tid       int64       `bson:"tid" json:"tid"`
	Signature string      `bson:"signature" json:"signature"`
	Epoch     schema.Date `bson:"epoch" json:"epoch"`
}

func NewSigningSignature() (SigningSignature, error) {
	const value = "{}"
	var object SigningSignature
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Container struct {
	WorkloadId        int64               `bson:"workload_id" json:"workload_id"`
	NodeId            string              `bson:"node_id" json:"node_id"`
	Flist             string              `bson:"flist" json:"flist"`
	HubUrl            string              `bson:"hub_url" json:"hub_url"`
	Environment       map[string]string   `bson:"environment" json:"environment"`
	SecretEnvironment map[string]string   `bson:"secret_environment" json:"secret_environment"`
	Entrypoint        string              `bson:"entrypoint" json:"entrypoint"`
	Interactive       bool                `bson:"interactive" json:"interactive"`
	Volumes           []ContainerMount    `bson:"volumes" json:"volumes"`
	NetworkConnection []NetworkConnection `bson:"network_connection" json:"network_connection"`
	StatsAggregator   []StatsAggregator   `bson:"stats_aggregator" json:"stats_aggregator"`
	Logs              []Logs              `bson:"logs" json:"logs"`
	FarmerTid         int64               `bson:"farmer_tid" json:"farmer_tid"`
	Capacity          ContainerCapacity   `bson:"capcity" json:"capacity"`
}

type ContainerCapacity struct {
	Cpu    int64 `bson:"cpu" json:"cpu"`
	Memory int64 `bson:"memory" json:"memory"`
}

type Logs struct {
	Type string    `bson:"type" json:"type"`
	Data LogsRedis `bson:"data" json:"data"`
}

type LogsRedis struct {
	Stdout string `bson:"stdout" json:"stdout"`
	Stderr string `bson:"stderr" json:"stderr"`
}

func NewContainer() (Container, error) {
	const value = "{\"interactive\": true}"
	var object Container
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type ContainerMount struct {
	VolumeId   string `bson:"volume_id" json:"volume_id"`
	Mountpoint string `bson:"mountpoint" json:"mountpoint"`
}

func NewTfgridWorkloadsReservationContainerMount1() (ContainerMount, error) {
	const value = "{}"
	var object ContainerMount
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type NetworkConnection struct {
	NetworkId string `bson:"network_id" json:"network_id"`
	Ipaddress net.IP `bson:"ipaddress" json:"ipaddress"`
	PublicIp6 bool   `bson:"public_ip6" json:"public_ip6"`
}

func NewNetworkConnection() (NetworkConnection, error) {
	const value = "{}"
	var object NetworkConnection
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type K8S struct {
	WorkloadId      int64             `bson:"workload_id" json:"workload_id"`
	NodeId          string            `bson:"node_id" json:"node_id"`
	Size            int64             `bson:"size" json:"size"`
	NetworkId       string            `bson:"network_id" json:"network_id"`
	Ipaddress       net.IP            `bson:"ipaddress" json:"ipaddress"`
	ClusterSecret   string            `bson:"cluster_secret" json:"cluster_secret"`
	MasterIps       []net.IP          `bson:"master_ips" json:"master_ips"`
	SshKeys         []string          `bson:"ssh_keys" json:"ssh_keys"`
	StatsAggregator []StatsAggregator `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64             `bson:"farmer_tid" json:"farmer_tid"`
}

func NewK8S() (K8S, error) {
	const value = "{}"
	var object K8S
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Network struct {
	Name             string               `bson:"name" json:"name"`
	WorkloadId       int64                `bson:"workload_id" json:"workload_id"`
	Iprange          schema.IPRange       `bson:"iprange" json:"iprange"`
	StatsAggregator  []StatsAggregator    `bson:"stats_aggregator" json:"stats_aggregator"`
	NetworkResources []NetworkNetResource `bson:"network_resources" json:"network_resources"`
	FarmerTid        int64                `bson:"farmer_tid" json:"farmer_tid"`
}

func NewNetwork() (Network, error) {
	const value = "{\"name\": \"\", \"iprange\": \"10.10.0.0/16\"}"
	var object Network
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type NetworkNetResource struct {
	NodeId                       string          `bson:"node_id" json:"node_id"`
	WireguardPrivateKeyEncrypted string          `bson:"wireguard_private_key_encrypted" json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string          `bson:"wireguard_public_key" json:"wireguard_public_key"`
	WireguardListenPort          int64           `bson:"wireguard_listen_port" json:"wireguard_listen_port"`
	Iprange                      schema.IPRange  `bson:"iprange" json:"iprange"`
	Peers                        []WireguardPeer `bson:"peers" json:"peers"`
}

func NewNetworkNetResource() (NetworkNetResource, error) {
	const value = "{\"wireguard_private_key_encrypted\": \"\", \"wireguard_public_key\": \"\", \"iprange\": \"10.10.10.0/24\"}"
	var object NetworkNetResource
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type WireguardPeer struct {
	PublicKey      string           `bson:"public_key" json:"public_key"`
	AllowedIprange []schema.IPRange `bson:"allowed_iprange" json:"allowed_iprange"`
	Endpoint       string           `bson:"endpoint" json:"endpoint"`
	Iprange        schema.IPRange   `bson:"iprange" json:"iprange"`
}

func NewPeer() (WireguardPeer, error) {
	const value = "{\"public_key\": \"\", \"allowed_iprange\": [], \"endpoint\": \"\", \"iprange\": \"10.10.11.0/24\"}"
	var object WireguardPeer
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Result struct {
	Category   ResultCategoryEnum `bson:"category" json:"category"`
	WorkloadId string             `bson:"workload_id" json:"workload_id"`
	DataJson   json.RawMessage    `bson:"data_json" json:"data_json"`
	Signature  string             `bson:"signature" json:"signature"`
	State      ResultStateEnum    `bson:"state" json:"state"`
	Message    string             `bson:"message" json:"message"`
	Epoch      schema.Date        `bson:"epoch" json:"epoch"`
	NodeId     string             `bson:"node_id" json:"node_id"`
}

func NewResult() (Result, error) {
	const value = "{\"message\": \"\"}"
	var object Result
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type StatsAggregator struct {
	Addr   string `bson:"addr" json:"addr"`
	Port   int64  `bson:"port" json:"port"`
	Secret string `bson:"secret" json:"secret"`
}

func NewStatsAggregator() (StatsAggregator, error) {
	const value = "{}"
	var object StatsAggregator
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Volume struct {
	WorkloadId      int64             `bson:"workload_id" json:"workload_id"`
	NodeId          string            `bson:"node_id" json:"node_id"`
	Size            int64             `bson:"size" json:"size"`
	Type            VolumeTypeEnum    `bson:"type" json:"type"`
	StatsAggregator []StatsAggregator `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64             `bson:"farmer_tid" json:"farmer_tid"`
}

func NewVolume() (Volume, error) {
	const value = "{}"
	var object Volume
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

// NOTE: this type has some manual changes
// that need to be preserved between regenerations.
type ReservationWorkload struct {
	WorkloadId string           `bson:"workload_id" json:"workload_id"`
	User       string           `bson:"user" json:"user"`
	Type       WorkloadTypeEnum `bson:"type" json:"type"`
	Content    interface{}      `bson:"content" json:"content"`
	Created    schema.Date      `bson:"created" json:"created"`
	Duration   int64            `bson:"duration" json:"duration"`
	Signature  string           `bson:"signature" json:"signature"`
	ToDelete   bool             `bson:"to_delete" json:"to_delete"`
}

func NewReservationWorkload() (ReservationWorkload, error) {
	const value = "{}"
	var object ReservationWorkload
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type ZDB struct {
	WorkloadId      int64             `bson:"workload_id" json:"workload_id"`
	NodeId          string            `bson:"node_id" json:"node_id"`
	Size            int64             `bson:"size" json:"size"`
	Mode            ZDBModeEnum       `bson:"mode" json:"mode"`
	Password        string            `bson:"password" json:"password"`
	DiskType        DiskTypeEnum      `bson:"disk_type" json:"disk_type"`
	Public          bool              `bson:"public" json:"public"`
	StatsAggregator []StatsAggregator `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64             `bson:"farmer_tid" json:"farmer_tid"`
}

func NewZDB() (ZDB, error) {
	const value = "{\"public\": false}"
	var object ZDB
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type DiskTypeEnum uint8

const (
	DiskTypeHDD DiskTypeEnum = iota
	DiskTypeSSD
)

func (e DiskTypeEnum) String() string {
	switch e {
	case DiskTypeHDD:
		return "hdd"
	case DiskTypeSSD:
		return "ssd"
	}
	return "UNKNOWN"
}

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

func (e NextActionEnum) String() string {
	switch e {
	case NextActionCreate:
		return "create"
	case NextActionSign:
		return "sign"
	case NextActionPay:
		return "pay"
	case NextActionDeploy:
		return "deploy"
	case NextActionDelete:
		return "delete"
	case NextActionInvalid:
		return "invalid"
	case NextActionDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}

type ResultCategoryEnum uint8

const (
	ResultCategoryZDB ResultCategoryEnum = iota
	ResultCategoryContainer
	ResultCategoryNetwork
	ResultCategoryVolume
	ResultCategoryK8S
)

func (e ResultCategoryEnum) String() string {
	switch e {
	case ResultCategoryZDB:
		return "zdb"
	case ResultCategoryContainer:
		return "container"
	case ResultCategoryNetwork:
		return "network"
	case ResultCategoryVolume:
		return "volume"
	case ResultCategoryK8S:
		return "kubernetes"
	}
	return "UNKNOWN"
}

type ResultStateEnum uint8

const (
	ResultStateError ResultStateEnum = iota
	ResultStateOK
	ResultStateDeleted
)

func (e ResultStateEnum) String() string {
	switch e {
	case ResultStateError:
		return "error"
	case ResultStateOK:
		return "ok"
	case ResultStateDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}

type VolumeTypeEnum uint8

const (
	VolumeTypeHDD VolumeTypeEnum = iota
	VolumeTypeSSD
)

func (e VolumeTypeEnum) String() string {
	switch e {
	case VolumeTypeHDD:
		return "HDD"
	case VolumeTypeSSD:
		return "SSD"
	}
	return "UNKNOWN"
}

type WorkloadTypeEnum uint8

const (
	WorkloadTypeZDB WorkloadTypeEnum = iota
	WorkloadTypeContainer
	WorkloadTypeVolume
	WorkloadTypeNetwork
	WorkloadTypeKubernetes
)

func (e WorkloadTypeEnum) String() string {
	switch e {
	case WorkloadTypeZDB:
		return "zdb"
	case WorkloadTypeContainer:
		return "container"
	case WorkloadTypeVolume:
		return "volume"
	case WorkloadTypeNetwork:
		return "network"
	case WorkloadTypeKubernetes:
		return "kubernetes"
	}
	return "UNKNOWN"
}

type ZDBModeEnum uint8

const (
	ZDBModeSeq ZDBModeEnum = iota
	ZDBModeUser
)

func (e ZDBModeEnum) String() string {
	switch e {
	case ZDBModeSeq:
		return "seq"
	case ZDBModeUser:
		return "user"
	}
	return "UNKNOWN"
}
