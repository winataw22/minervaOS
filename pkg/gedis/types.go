package gedis

import (
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

// 'structNameWithoutPrefix' are internal struct used
// 'gedisStructNameBlabla' are gedis datatype wrappers

type registerNodeBody struct {
	NodeID  string `json:"node_id,omitempty"`
	FarmID  string `json:"farm_id,omitempty"`
	Version string `json:"os_version"`
}

type gedisRegisterNodeBody struct {
	Node tfgridNode2 `json:"node"`
}

//
// generated with schemac from model
//
type tfgridLocation1 struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Continent string  `json:"continent"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type tfgridNodeResource1 struct {
	Cru int64 `json:"cru"`
	Mru int64 `json:"mru"`
	Hru int64 `json:"hru"`
	Sru int64 `json:"sru"`
}

type getNodeBody struct {
	NodeID string `json:"node_id,omitempty"`
}

type gedisListNodeBodyPayload struct {
	FarmID  string `json:"farmer_id"`
	Country string `json:"country"`
	City    string `json:"city"`
	Cru     int    `json:",omitempty"`
	Mru     int    `json:",omitempty"`
	Sru     int    `json:",omitempty"`
	Hru     int    `json:",omitempty"`
}

type gedisListNodeResponseBody struct {
	Nodes []tfgridNode2 `json:"nodes"`
}

type tfgridNodePublicIface1TypeEnum uint8

const (
	tfgridNodePublicIface1Type tfgridNodePublicIface1TypeEnum = iota
)

type tfgridNodeIface1 struct {
	Name    string           `json:"name"`
	Addrs   []schema.IPRange `json:"addrs"`
	Gateway []net.IP         `json:"gateway"`
}

type tfgridNodePublicIface1 struct {
	Master  string                         `json:"master"`
	Type    tfgridNodePublicIface1TypeEnum `json:"type"`
	Ipv4    net.IP                         `json:"ipv4"`
	Ipv6    net.IP                         `json:"ipv6"`
	Gw4     net.IP                         `json:"gw4"`
	Gw6     net.IP                         `json:"gw6"`
	Version int64                          `json:"version"`
}

type tfgridFarm1 struct {
	ThreebotID      string              `json:"threebot_id"`
	IyoOrganization string              `json:"iyo_organization"`
	Name            string              `json:"name"`
	WalletAddresses []string            `json:"wallet_addresses"`
	Location        tfgridLocation1     `json:"location"`
	Vta             string              `json:"vta"`
	ResourcePrice   tfgridNodeResource1 `json:"resource_price"`
}

type tfgridNode2 struct {
	NodeID           string                 `json:"node_id,omitempty"`
	FarmID           string                 `json:"farmer_id,omitempty"`
	Version          string                 `json:"os_version"`
	Uptime           int64                  `json:"uptime"`
	Address          string                 `json:"address"`
	Location         tfgridLocation1        `json:"location"`
	TotalResource    tfgridNodeResource1    `json:"total_resource"`
	UsedResource     tfgridNodeResource1    `json:"used_resource"`
	ReservedResource tfgridNodeResource1    `json:"reserved_resource"`
	Ifaces           []tfgridNodeIface1     `json:"ifaces"`
	PublicConfig     tfgridNodePublicIface1 `json:"public_config"`
	ExitNode         bool                   `json:"exit_node"`
	Approved         bool                   `json:"approved"`
}

type gedisNodeUpdateCapacity struct {
	NodeID   string              `json:"node_id"`
	Resource tfgridNodeResource1 `json:"resource"`
}

//
// Farms
//

type registerFarmBody struct {
	Farm string `json:"farm_id,omitempty"`
	Name string `json:"name,omitempty"`
}

type gedisRegisterFarmBody struct {
	Farm gedisRegisterFarmBodyPayload `json:"farm"`
}

type gedisRegisterFarmBodyPayload struct {
	ThreebotID string   `json:"threebot_id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Email      string   `json:"email,omitempty"`
	Wallet     []string `json:"wallet_addresses"`
}

type getFarmBody struct {
	FarmID string `json:"farm_id,omitempty"`
}

type gedisGetFarmBody struct {
	Farm tfgridFarm1 `json:"farm"`
}

type gedisUpdateFarmBody struct {
	FarmID string      `json:"farm_id"`
	Farm   tfgridFarm1 `json:"farm"`
}

type gedisListFarmBody struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

type gedisListFarmResponseBody struct {
	Farms []tfgridFarm1 `json:"farms"`
}
