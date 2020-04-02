package directory

import (
	"encoding/json"
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

type Farm struct {
	ID              schema.ID           `bson:"_id" json:"id"`
	ThreebotId      int64               `bson:"threebot_id" json:"threebot_id"`
	IyoOrganization string              `bson:"iyo_organization" json:"iyo_organization"`
	Name            string              `bson:"name" json:"name"`
	WalletAddresses []WalletAddress     `bson:"wallet_addresses" json:"wallet_addresses"`
	Location        Location            `bson:"location" json:"location"`
	Email           schema.Email        `bson:"email" json:"email"`
	ResourcePrices  []NodeResourcePrice `bson:"resource_prices" json:"resource_prices"`
	PrefixZero      schema.IPRange      `bson:"prefix_zero" json:"prefix_zero"`
}

func NewFarm() (Farm, error) {
	const value = "{}"
	var object Farm
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type WalletAddress struct {
	Asset   string `bson:"asset" json:"asset"`
	Address string `bson:"address" json:"address"`
}

type NodeResourcePrice struct {
	Currency PriceCurrencyEnum `bson:"currency" json:"currency"`
	Cru      float64           `bson:"cru" json:"cru"`
	Mru      float64           `bson:"mru" json:"mru"`
	Hru      float64           `bson:"hru" json:"hru"`
	Sru      float64           `bson:"sru" json:"sru"`
	Nru      float64           `bson:"nru" json:"nru"`
}

func NewNodeResourcePrice() (NodeResourcePrice, error) {
	const value = "{}"
	var object NodeResourcePrice
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Location struct {
	City      string  `bson:"city" json:"city"`
	Country   string  `bson:"country" json:"country"`
	Continent string  `bson:"continent" json:"continent"`
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}

func NewLocation() (Location, error) {
	const value = "{}"
	var object Location
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Node struct {
	ID                schema.ID      `bson:"_id" json:"id"`
	NodeId            string         `bson:"node_id" json:"node_id"`
	NodeIdV1          string         `bson:"node_id_v1" json:"node_id_v1"`
	FarmId            int64          `bson:"farm_id" json:"farm_id"`
	OsVersion         string         `bson:"os_version" json:"os_version"`
	Created           schema.Date    `bson:"created" json:"created"`
	Updated           schema.Date    `bson:"updated" json:"updated"`
	Uptime            int64          `bson:"uptime" json:"uptime"`
	Address           string         `bson:"address" json:"address"`
	Location          Location       `bson:"location" json:"location"`
	TotalResources    ResourceAmount `bson:"total_resources" json:"total_resources"`
	UsedResources     ResourceAmount `bson:"used_resources" json:"used_resources"`
	ReservedResources ResourceAmount `bson:"reserved_resources" json:"reserved_resources"`
	Proofs            []Proof        `bson:"proofs" json:"proofs"`
	Ifaces            []Iface        `bson:"ifaces" json:"ifaces"`
	PublicConfig      *PublicIface   `bson:"public_config,omitempty" json:"public_config"`
	FreeToUse         bool           `bson:"free_to_use" json:"free_to_use"`
	Approved          bool           `bson:"approved" json:"approved"`
	PublicKeyHex      string         `bson:"public_key_hex" json:"public_key_hex"`
	WgPorts           []int64        `bson:"wg_ports" json:"wg_ports"`
}

func NewNode() (Node, error) {
	const value = "{\"approved\": false, \"public_key_hex\": \"\"}"
	var object Node
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Iface struct {
	Name       string            `bson:"name" json:"name"`
	Addrs      []schema.IPRange  `bson:"addrs" json:"addrs"`
	Gateway    []net.IP          `bson:"gateway" json:"gateway"`
	MacAddress schema.MacAddress `bson:"macaddress" json:"macaddress"`
}

func NewIface() (Iface, error) {
	const value = "{}"
	var object Iface
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type PublicIface struct {
	Master  string         `bson:"master" json:"master"`
	Type    IfaceTypeEnum  `bson:"type" json:"type"`
	Ipv4    schema.IPRange `bson:"ipv4" json:"ipv4"`
	Ipv6    schema.IPRange `bson:"ipv6" json:"ipv6"`
	Gw4     net.IP         `bson:"gw4" json:"gw4"`
	Gw6     net.IP         `bson:"gw6" json:"gw6"`
	Version int64          `bson:"version" json:"version"`
}

func NewPublicIface() (PublicIface, error) {
	const value = "{}"
	var object PublicIface
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type ResourceAmount struct {
	Cru int64 `bson:"cru" json:"cru"`
	Mru int64 `bson:"mru" json:"mru"`
	Hru int64 `bson:"hru" json:"hru"`
	Sru int64 `bson:"sru" json:"sru"`
}

func NewResourceAmount() (ResourceAmount, error) {
	const value = "{}"
	var object ResourceAmount
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type Proof struct {
	Created      schema.Date            `bson:"created" json:"created"`
	HardwareHash string                 `bson:"hardware_hash" json:"hardware_hash"`
	DiskHash     string                 `bson:"disk_hash" json:"disk_hash"`
	Hardware     map[string]interface{} `bson:"hardware" json:"hardware"`
	Disks        map[string]interface{} `bson:"disks" json:"disks"`
	Hypervisor   []string               `bson:"hypervisor" json:"hypervisor"`
}

func NewProof() (Proof, error) {
	const value = "{}"
	var object Proof
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type IfaceTypeEnum uint8

const (
	IfaceTypeMacvlan IfaceTypeEnum = iota
	IfaceTypeVlan
)

func (e IfaceTypeEnum) String() string {
	switch e {
	case IfaceTypeMacvlan:
		return "macvlan"
	case IfaceTypeVlan:
		return "vlan"
	}
	return "UNKNOWN"
}

type PriceCurrencyEnum uint8

const (
	PriceCurrencyEUR PriceCurrencyEnum = iota
	PriceCurrencyUSD
	PriceCurrencyTFT
	PriceCurrencyAED
	PriceCurrencyGBP
)

func (e PriceCurrencyEnum) String() string {
	switch e {
	case PriceCurrencyEUR:
		return "EUR"
	case PriceCurrencyUSD:
		return "USD"
	case PriceCurrencyTFT:
		return "TFT"
	case PriceCurrencyAED:
		return "AED"
	case PriceCurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}
