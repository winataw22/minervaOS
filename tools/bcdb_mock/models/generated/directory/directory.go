package directory

import (
	"encoding/json"
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridDirectoryFarm1 struct {
	ID              schema.ID                           `bson:"_id" json:"id"`
	ThreebotId      int64                               `bson:"threebot_id" json:"threebot_id"`
	IyoOrganization string                              `bson:"iyo_organization" json:"iyo_organization"`
	Name            string                              `bson:"name" json:"name"`
	WalletAddresses []string                            `bson:"wallet_addresses" json:"wallet_addresses"`
	Location        TfgridDirectoryLocation1            `bson:"location" json:"location"`
	Email           schema.Email                        `bson:"email" json:"email"`
	ResourcePrices  []TfgridDirectoryNodeResourcePrice1 `bson:"resource_prices" json:"resource_prices"`
	PrefixZero      schema.IPRange                      `bson:"prefix_zero" json:"prefix_zero"`
}

func NewTfgridDirectoryFarm1() (TfgridDirectoryFarm1, error) {
	const value = "{}"
	var object TfgridDirectoryFarm1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeResourcePrice1 struct {
	Currency TfgridDirectoryNodeResourcePrice1CurrencyEnum `bson:"currency" json:"currency"`
	Cru      float64                                       `bson:"cru" json:"cru"`
	Mru      float64                                       `bson:"mru" json:"mru"`
	Hru      float64                                       `bson:"hru" json:"hru"`
	Sru      float64                                       `bson:"sru" json:"sru"`
	Nru      float64                                       `bson:"nru" json:"nru"`
}

func NewTfgridDirectoryNodeResourcePrice1() (TfgridDirectoryNodeResourcePrice1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeResourcePrice1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryLocation1 struct {
	City      string  `bson:"city" json:"city"`
	Country   string  `bson:"country" json:"country"`
	Continent string  `bson:"continent" json:"continent"`
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}

func NewTfgridDirectoryLocation1() (TfgridDirectoryLocation1, error) {
	const value = "{}"
	var object TfgridDirectoryLocation1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNode2 struct {
	ID                schema.ID                          `bson:"_id" json:"id"`
	NodeId            string                             `bson:"node_id" json:"node_id"`
	NodeIdV1          string                             `bson:"node_id_v1" json:"node_id_v1"`
	FarmId            int64                              `bson:"farm_id" json:"farm_id"`
	OsVersion         string                             `bson:"os_version" json:"os_version"`
	Created           schema.Date                        `bson:"created" json:"created"`
	Updated           schema.Date                        `bson:"updated" json:"updated"`
	Uptime            int64                              `bson:"uptime" json:"uptime"`
	Address           string                             `bson:"address" json:"address"`
	Location          TfgridDirectoryLocation1           `bson:"location" json:"location"`
	TotalResources    TfgridDirectoryNodeResourceAmount1 `bson:"total_resources" json:"total_resources"`
	UsedResources     TfgridDirectoryNodeResourceAmount1 `bson:"used_resources" json:"used_resources"`
	ReservedResources TfgridDirectoryNodeResourceAmount1 `bson:"reserved_resources" json:"reserved_resources"`
	Proofs            []TfgridDirectoryNodeProof1        `bson:"proofs" json:"proofs"`
	Ifaces            []TfgridDirectoryNodeIface1        `bson:"ifaces" json:"ifaces"`
	PublicConfig      *TfgridDirectoryNodePublicIface1   `bson:"public_config" json:"public_config"`
	ExitNode          bool                               `bson:"exit_node" json:"exit_node"`
	Approved          bool                               `bson:"approved" json:"approved"`
	PublicKeyHex      string                             `bson:"public_key_hex" json:"public_key_hex"`
	WgPorts           []int64                            `bson:"wg_ports" json:"wg_ports"`
}

func NewTfgridDirectoryNode2() (TfgridDirectoryNode2, error) {
	const value = "{\"approved\": false, \"public_key_hex\": \"\"}"
	var object TfgridDirectoryNode2
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeIface1 struct {
	Name    string           `bson:"name" json:"name"`
	Addrs   []schema.IPRange `bson:"addrs" json:"addrs"`
	Gateway []net.IP         `bson:"gateway" json:"gateway"`
}

func NewTfgridDirectoryNodeIface1() (TfgridDirectoryNodeIface1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodePublicIface1 struct {
	Master  string                                  `bson:"master" json:"master"`
	Type    TfgridDirectoryNodePublicIface1TypeEnum `bson:"type" json:"type"`
	Ipv4    schema.IPRange                          `bson:"ipv4" json:"ipv4"`
	Ipv6    schema.IPRange                          `bson:"ipv6" json:"ipv6"`
	Gw4     net.IP                                  `bson:"gw4" json:"gw4"`
	Gw6     net.IP                                  `bson:"gw6" json:"gw6"`
	Version int64                                   `bson:"version" json:"version"`
}

func NewTfgridDirectoryNodePublicIface1() (TfgridDirectoryNodePublicIface1, error) {
	const value = "{}"
	var object TfgridDirectoryNodePublicIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeResourceAmount1 struct {
	Cru int64 `bson:"cru" json:"cru"`
	Mru int64 `bson:"mru" json:"mru"`
	Hru int64 `bson:"hru" json:"hru"`
	Sru int64 `bson:"sru" json:"sru"`
}

func NewTfgridDirectoryNodeResourceAmount1() (TfgridDirectoryNodeResourceAmount1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeResourceAmount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeProof1 struct {
	Created      schema.Date            `bson:"created" json:"created"`
	HardwareHash string                 `bson:"hardware_hash" json:"hardware_hash"`
	DiskHash     string                 `bson:"disk_hash" json:"disk_hash"`
	Hardware     map[string]interface{} `bson:"hardware" json:"hardware"`
	Disks        map[string]interface{} `bson:"disks" json:"disks"`
	Hypervisor   []string               `bson:"hypervisor" json:"hypervisor"`
}

func NewTfgridDirectoryNodeProof1() (TfgridDirectoryNodeProof1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeProof1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodePublicIface1TypeEnum uint8

const (
	TfgridDirectoryNodePublicIface1TypeMacvlan TfgridDirectoryNodePublicIface1TypeEnum = iota
	TfgridDirectoryNodePublicIface1TypeVlan
)

func (e TfgridDirectoryNodePublicIface1TypeEnum) String() string {
	switch e {
	case TfgridDirectoryNodePublicIface1TypeMacvlan:
		return "macvlan"
	case TfgridDirectoryNodePublicIface1TypeVlan:
		return "vlan"
	}
	return "UNKNOWN"
}

type TfgridDirectoryNodeResourcePrice1CurrencyEnum uint8

const (
	TfgridDirectoryNodeResourcePrice1CurrencyEUR TfgridDirectoryNodeResourcePrice1CurrencyEnum = iota
	TfgridDirectoryNodeResourcePrice1CurrencyUSD
	TfgridDirectoryNodeResourcePrice1CurrencyTFT
	TfgridDirectoryNodeResourcePrice1CurrencyAED
	TfgridDirectoryNodeResourcePrice1CurrencyGBP
)

func (e TfgridDirectoryNodeResourcePrice1CurrencyEnum) String() string {
	switch e {
	case TfgridDirectoryNodeResourcePrice1CurrencyEUR:
		return "EUR"
	case TfgridDirectoryNodeResourcePrice1CurrencyUSD:
		return "USD"
	case TfgridDirectoryNodeResourcePrice1CurrencyTFT:
		return "TFT"
	case TfgridDirectoryNodeResourcePrice1CurrencyAED:
		return "AED"
	case TfgridDirectoryNodeResourcePrice1CurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}
