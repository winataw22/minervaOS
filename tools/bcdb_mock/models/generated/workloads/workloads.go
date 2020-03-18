package workloads

import (
	"encoding/json"
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridWorkloadsReservation1 struct {
	ID                  schema.ID                                     `bson:"_id" json:"id"`
	Json                json.RawMessage                               `bson:"json" json:"json"`
	DataReservation     TfgridWorkloadsReservationData1               `bson:"data_reservation" json:"data_reservation"`
	CustomerTid         int64                                         `bson:"customer_tid" json:"customer_tid"`
	CustomerSignature   string                                        `bson:"customer_signature" json:"customer_signature"`
	NextAction          TfgridWorkloadsReservation1NextActionEnum     `bson:"next_action" json:"next_action"`
	SignaturesProvision []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_provision" json:"signatures_provision"`
	SignaturesFarmer    []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_farmer" json:"signatures_farmer"`
	SignaturesDelete    []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_delete" json:"signatures_delete"`
	Epoch               schema.Date                                   `bson:"epoch" json:"epoch"`
	Results             []TfgridWorkloadsReservationResult1           `bson:"results" json:"results"`
}

func NewTfgridWorkloadsReservation1() (TfgridWorkloadsReservation1, error) {
	const value = "{\"json\": \"\"}"
	var object TfgridWorkloadsReservation1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationData1 struct {
	Description             string                                    `bson:"description" json:"description"`
	SigningRequestProvision TfgridWorkloadsReservationSigningRequest1 `bson:"signing_request_provision" json:"signing_request_provision"`
	SigningRequestDelete    TfgridWorkloadsReservationSigningRequest1 `bson:"signing_request_delete" json:"signing_request_delete"`
	Containers              []TfgridWorkloadsReservationContainer1    `bson:"containers" json:"containers"`
	Volumes                 []TfgridWorkloadsReservationVolume1       `bson:"volumes" json:"volumes"`
	Zdbs                    []TfgridWorkloadsReservationZdb1          `bson:"zdbs" json:"zdbs"`
	Networks                []TfgridWorkloadsReservationNetwork1      `bson:"networks" json:"networks"`
	Kubernetes              []TfgridWorkloadsReservationK8S1          `bson:"kubernetes" json:"kubernetes"`
	ExpirationProvisioning  schema.Date                               `bson:"expiration_provisioning" json:"expiration_provisioning"`
	ExpirationReservation   schema.Date                               `bson:"expiration_reservation" json:"expiration_reservation"`
}

func NewTfgridWorkloadsReservationData1() (TfgridWorkloadsReservationData1, error) {
	const value = "{\"description\": \"\"}"
	var object TfgridWorkloadsReservationData1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationSigningRequest1 struct {
	Signers   []int64 `bson:"signers" json:"signers"`
	QuorumMin int64   `bson:"quorum_min" json:"quorum_min"`
}

func NewTfgridWorkloadsReservationSigningRequest1() (TfgridWorkloadsReservationSigningRequest1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationSigningRequest1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationSigningSignature1 struct {
	Tid       int64       `bson:"tid" json:"tid"`
	Signature string      `bson:"signature" json:"signature"`
	Epoch     schema.Date `bson:"epoch" json:"epoch"`
}

func NewTfgridWorkloadsReservationSigningSignature1() (TfgridWorkloadsReservationSigningSignature1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationSigningSignature1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationContainer1 struct {
	WorkloadId        int64                                          `bson:"workload_id" json:"workload_id"`
	NodeId            string                                         `bson:"node_id" json:"node_id"`
	Flist             string                                         `bson:"flist" json:"flist"`
	HubUrl            string                                         `bson:"hub_url" json:"hub_url"`
	Environment       map[string]interface{}                         `bson:"environment" json:"environment"`
	SecretEnvironment map[string]interface{}                         `bson:"secret_environment" json:"secret_environment"`
	Entrypoint        string                                         `bson:"entrypoint" json:"entrypoint"`
	Interactive       bool                                           `bson:"interactive" json:"interactive"`
	Volumes           []TfgridWorkloadsReservationContainerMount1    `bson:"volumes" json:"volumes"`
	NetworkConnection []TfgridWorkloadsReservationNetworkConnection1 `bson:"network_connection" json:"network_connection"`
	StatsAggregator   []TfgridWorkloadsReservationStatsaggregator1   `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid         int64                                          `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationContainer1() (TfgridWorkloadsReservationContainer1, error) {
	const value = "{\"interactive\": true}"
	var object TfgridWorkloadsReservationContainer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationContainerMount1 struct {
	VolumeId   string `bson:"volume_id" json:"volume_id"`
	Mountpoint string `bson:"mountpoint" json:"mountpoint"`
}

func NewTfgridWorkloadsReservationContainerMount1() (TfgridWorkloadsReservationContainerMount1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationContainerMount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationNetworkConnection1 struct {
	NetworkId string `bson:"network_id" json:"network_id"`
	Ipaddress net.IP `bson:"ipaddress" json:"ipaddress"`
}

func NewTfgridWorkloadsReservationNetworkConnection1() (TfgridWorkloadsReservationNetworkConnection1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationNetworkConnection1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationK8S1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	NetworkId       string                                       `bson:"network_id" json:"network_id"`
	Ipaddress       net.IP                                       `bson:"ipaddress" json:"ipaddress"`
	ClusterSecret   string                                       `bson:"cluster_secret" json:"cluster_secret"`
	MasterIps       []net.IP                                     `bson:"master_ips" json:"master_ips"`
	SshKeys         []string                                     `bson:"ssh_keys" json:"ssh_keys"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationK8S1() (TfgridWorkloadsReservationK8S1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationK8S1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationNetwork1 struct {
	Name             string                                       `bson:"name" json:"name"`
	WorkloadId       int64                                        `bson:"workload_id" json:"workload_id"`
	Iprange          schema.IPRange                               `bson:"iprange" json:"iprange"`
	StatsAggregator  []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	NetworkResources []TfgridWorkloadsNetworkNetResource1         `bson:"network_resources" json:"network_resources"`
	FarmerTid        int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationNetwork1() (TfgridWorkloadsReservationNetwork1, error) {
	const value = "{\"name\": \"\", \"iprange\": \"10.10.0.0/16\"}"
	var object TfgridWorkloadsReservationNetwork1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsNetworkNetResource1 struct {
	NodeId                       string                          `bson:"node_id" json:"node_id"`
	WireguardPrivateKeyEncrypted string                          `bson:"wireguard_private_key_encrypted" json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string                          `bson:"wireguard_public_key" json:"wireguard_public_key"`
	WireguardListenPort          int64                           `bson:"wireguard_listen_port" json:"wireguard_listen_port"`
	Iprange                      schema.IPRange                  `bson:"iprange" json:"iprange"`
	Peers                        []TfgridWorkloadsWireguardPeer1 `bson:"peers" json:"peers"`
}

func NewTfgridWorkloadsNetworkNetResource1() (TfgridWorkloadsNetworkNetResource1, error) {
	const value = "{\"wireguard_private_key_encrypted\": \"\", \"wireguard_public_key\": \"\", \"iprange\": \"10.10.10.0/24\"}"
	var object TfgridWorkloadsNetworkNetResource1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsWireguardPeer1 struct {
	PublicKey      string           `bson:"public_key" json:"public_key"`
	AllowedIprange []schema.IPRange `bson:"allowed_iprange" json:"allowed_iprange"`
	Endpoint       string           `bson:"endpoint" json:"endpoint"`
	Iprange        schema.IPRange   `bson:"iprange" json:"iprange"`
}

func NewTfgridWorkloadsWireguardPeer1() (TfgridWorkloadsWireguardPeer1, error) {
	const value = "{\"public_key\": \"\", \"allowed_iprange\": [], \"endpoint\": \"\", \"iprange\": \"10.10.11.0/24\"}"
	var object TfgridWorkloadsWireguardPeer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationResult1 struct {
	Category   TfgridWorkloadsReservationResult1CategoryEnum `bson:"category" json:"category"`
	WorkloadId string                                        `bson:"workload_id" json:"workload_id"`
	DataJson   json.RawMessage                               `bson:"data_json" json:"data_json"`
	Signature  string                                        `bson:"signature" json:"signature"`
	State      TfgridWorkloadsReservationResult1StateEnum    `bson:"state" json:"state"`
	Message    string                                        `bson:"message" json:"message"`
	Epoch      schema.Date                                   `bson:"epoch" json:"epoch"`
	NodeId     string                                        `bson:"node_id" json:"node_id"`
}

func NewTfgridWorkloadsReservationResult1() (TfgridWorkloadsReservationResult1, error) {
	const value = "{\"message\": \"\"}"
	var object TfgridWorkloadsReservationResult1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationStatsaggregator1 struct {
	Addr   string `bson:"addr" json:"addr"`
	Port   int64  `bson:"port" json:"port"`
	Secret string `bson:"secret" json:"secret"`
}

func NewTfgridWorkloadsReservationStatsaggregator1() (TfgridWorkloadsReservationStatsaggregator1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationStatsaggregator1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationVolume1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	Type            TfgridWorkloadsReservationVolume1TypeEnum    `bson:"type" json:"type"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationVolume1() (TfgridWorkloadsReservationVolume1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationVolume1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

// NOTE: this type has some manual changes
// that need to be preserved between regenerations.
type TfgridWorkloadsReservationWorkload1 struct {
	WorkloadId string                                      `bson:"workload_id" json:"workload_id"`
	User       string                                      `bson:"user" json:"user"`
	Type       TfgridWorkloadsReservationWorkload1TypeEnum `bson:"type" json:"type"`
	Content    interface{}                                 `bson:"content" json:"content"`
	Created    schema.Date                                 `bson:"created" json:"created"`
	Duration   int64                                       `bson:"duration" json:"duration"`
	Signature  string                                      `bson:"signature" json:"signature"`
	ToDelete   bool                                        `bson:"to_delete" json:"to_delete"`
}

func NewTfgridWorkloadsReservationWorkload1() (TfgridWorkloadsReservationWorkload1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationWorkload1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationZdb1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	Mode            TfgridWorkloadsReservationZdb1ModeEnum       `bson:"mode" json:"mode"`
	Password        string                                       `bson:"password" json:"password"`
	DiskType        TfgridWorkloadsReservationZdb1DiskTypeEnum   `bson:"disk_type" json:"disk_type"`
	Public          bool                                         `bson:"public" json:"public"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationZdb1() (TfgridWorkloadsReservationZdb1, error) {
	const value = "{\"public\": false}"
	var object TfgridWorkloadsReservationZdb1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationZdb1DiskTypeEnum uint8

const (
	TfgridWorkloadsReservationZdb1DiskTypeHdd TfgridWorkloadsReservationZdb1DiskTypeEnum = iota
	TfgridWorkloadsReservationZdb1DiskTypeSsd
)

func (e TfgridWorkloadsReservationZdb1DiskTypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationZdb1DiskTypeHdd:
		return "hdd"
	case TfgridWorkloadsReservationZdb1DiskTypeSsd:
		return "ssd"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservation1NextActionEnum uint8

const (
	TfgridWorkloadsReservation1NextActionCreate TfgridWorkloadsReservation1NextActionEnum = iota
	TfgridWorkloadsReservation1NextActionSign
	TfgridWorkloadsReservation1NextActionPay
	TfgridWorkloadsReservation1NextActionDeploy
	TfgridWorkloadsReservation1NextActionDelete
	TfgridWorkloadsReservation1NextActionInvalid
	TfgridWorkloadsReservation1NextActionDeleted
)

func (e TfgridWorkloadsReservation1NextActionEnum) String() string {
	switch e {
	case TfgridWorkloadsReservation1NextActionCreate:
		return "create"
	case TfgridWorkloadsReservation1NextActionSign:
		return "sign"
	case TfgridWorkloadsReservation1NextActionPay:
		return "pay"
	case TfgridWorkloadsReservation1NextActionDeploy:
		return "deploy"
	case TfgridWorkloadsReservation1NextActionDelete:
		return "delete"
	case TfgridWorkloadsReservation1NextActionInvalid:
		return "invalid"
	case TfgridWorkloadsReservation1NextActionDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationResult1CategoryEnum uint8

const (
	TfgridWorkloadsReservationResult1CategoryZdb TfgridWorkloadsReservationResult1CategoryEnum = iota
	TfgridWorkloadsReservationResult1CategoryContainer
	TfgridWorkloadsReservationResult1CategoryNetwork
	TfgridWorkloadsReservationResult1CategoryVolume
)

func (e TfgridWorkloadsReservationResult1CategoryEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationResult1CategoryZdb:
		return "zdb"
	case TfgridWorkloadsReservationResult1CategoryContainer:
		return "container"
	case TfgridWorkloadsReservationResult1CategoryNetwork:
		return "network"
	case TfgridWorkloadsReservationResult1CategoryVolume:
		return "volume"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationResult1StateEnum uint8

const (
	TfgridWorkloadsReservationResult1StateError TfgridWorkloadsReservationResult1StateEnum = iota
	TfgridWorkloadsReservationResult1StateOk
	TfgridWorkloadsReservationResult1StateDeleted
)

func (e TfgridWorkloadsReservationResult1StateEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationResult1StateError:
		return "error"
	case TfgridWorkloadsReservationResult1StateOk:
		return "ok"
	case TfgridWorkloadsReservationResult1StateDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationVolume1TypeEnum uint8

const (
	TfgridWorkloadsReservationVolume1TypeHDD TfgridWorkloadsReservationVolume1TypeEnum = iota
	TfgridWorkloadsReservationVolume1TypeSSD
)

func (e TfgridWorkloadsReservationVolume1TypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationVolume1TypeHDD:
		return "HDD"
	case TfgridWorkloadsReservationVolume1TypeSSD:
		return "SSD"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationWorkload1TypeEnum uint8

const (
	TfgridWorkloadsReservationWorkload1TypeZdb TfgridWorkloadsReservationWorkload1TypeEnum = iota
	TfgridWorkloadsReservationWorkload1TypeContainer
	TfgridWorkloadsReservationWorkload1TypeVolume
	TfgridWorkloadsReservationWorkload1TypeNetwork
	TfgridWorkloadsReservationWorkload1TypeKubernetes
)

func (e TfgridWorkloadsReservationWorkload1TypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationWorkload1TypeZdb:
		return "zdb"
	case TfgridWorkloadsReservationWorkload1TypeContainer:
		return "container"
	case TfgridWorkloadsReservationWorkload1TypeVolume:
		return "volume"
	case TfgridWorkloadsReservationWorkload1TypeNetwork:
		return "network"
	case TfgridWorkloadsReservationWorkload1TypeKubernetes:
		return "kubernetes"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationZdb1ModeEnum uint8

const (
	TfgridWorkloadsReservationZdb1ModeSeq TfgridWorkloadsReservationZdb1ModeEnum = iota
	TfgridWorkloadsReservationZdb1ModeUser
)

func (e TfgridWorkloadsReservationZdb1ModeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationZdb1ModeSeq:
		return "seq"
	case TfgridWorkloadsReservationZdb1ModeUser:
		return "user"
	}
	return "UNKNOWN"
}
