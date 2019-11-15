package directory

import (
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

//TfgridNode2 jsx schema
type TfgridNode2 struct {
	NodeID            string                    `json:"node_id"`
	NodeIDv1          string                    `json:"node_id_v1"`
	FarmID            uint64                    `json:"farm_id"`
	OsVersion         string                    `json:"os_version"`
	Created           schema.Date               `json:"created"`
	Updated           schema.Date               `json:"updated"`
	Uptime            int64                     `json:"uptime"`
	Address           string                    `json:"address"`
	Location          TfgridLocation1           `json:"location"`
	TotalResources    TfgridNodeResourceAmount1 `json:"total_resources"`
	UsedResources     TfgridNodeResourceAmount1 `json:"used_resources"`
	ReservedResources TfgridNodeResourceAmount1 `json:"reserved_resources"`
	Proofs            []TfgridNodeProof1        `json:"proofs"`
	Ifaces            []TfgridNodeIface1        `json:"ifaces"`
	PublicConfig      *TfgridNodePublicIface1   `json:"public_config,omitemtpy"`
	WGPorts           []uint                    `json:"wg_ports"`
	Approved          bool                      `json:"approved"`
	PublicKeyHex      string                    `json:"public_key_hex"`
}

// TfgridNodeIface1 jsx schema
type TfgridNodeIface1 struct {
	Name    string           `json:"name"`
	Addrs   []schema.IPRange `json:"addrs"`
	Gateway []net.IP         `json:"gateway"`
}

// TfgridNodePublicIface1 jsx schema
type TfgridNodePublicIface1 struct {
	Master  string                         `json:"master"`
	Type    TfgridNodePublicIface1TypeEnum `json:"type"`
	Ipv4    schema.IPRange                 `json:"ipv4"`
	Ipv6    schema.IPRange                 `json:"ipv6"`
	Gw4     net.IP                         `json:"gw4"`
	Gw6     net.IP                         `json:"gw6"`
	Version int64                          `json:"version"`
}

// TfgridNodeResourceAmount1 jsx schema
type TfgridNodeResourceAmount1 struct {
	Cru int64 `json:"cru"`
	Mru int64 `json:"mru"`
	Hru int64 `json:"hru"`
	Sru int64 `json:"sru"`
}

// TfgridNodeProof1 jsx schema
type TfgridNodeProof1 struct {
	Created      schema.Date            `json:"created"`
	HardwareHash string                 `json:"hardware_hash"`
	DiskHash     string                 `json:"disk_hash"`
	Hardware     map[string]interface{} `json:"hardware"`
	Disks        map[string]interface{} `json:"disks"`
}

// Equal test of 2 proofs have the same hashes
func (p TfgridNodeProof1) Equal(proof TfgridNodeProof1) bool {
	return p.DiskHash == proof.DiskHash && p.HardwareHash == proof.HardwareHash
}

// TfgridNodePublicIface1TypeEnum jsx schema
type TfgridNodePublicIface1TypeEnum uint8

// TfgridNodePublicIface1TypeEnum
const (
	TfgridNodePublicIface1TypeMacvlan TfgridNodePublicIface1TypeEnum = iota
	TfgridNodePublicIface1TypeVlan
)

// String implements stringer interface
func (e TfgridNodePublicIface1TypeEnum) String() string {
	switch e {
	case TfgridNodePublicIface1TypeMacvlan:
		return "macvlan"
	case TfgridNodePublicIface1TypeVlan:
		return "vlan"
	}
	return "UNKNOWN"
}
