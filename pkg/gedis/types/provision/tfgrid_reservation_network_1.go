package provision

import (
	schema "github.com/threefoldtech/zos/pkg/schema"
)

// TfgridReservationNetwork1 jsx Schema
type TfgridReservationNetwork1 struct {
	Name             string                              `json:"name"`
	WorkloadID       int64                               `json:"workload_id"`
	Iprange          schema.IPRange                      `json:"iprange"`
	StatsAggregator  []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	NetworkResources []TfgridNetworkNetResource1         `json:"network_resources"`
}

// TfgridNetworkNetResource1 jsx Schema
type TfgridNetworkNetResource1 struct {
	NodeID                       string           `json:"node_id"`
	IPRange                      schema.IPRange   `json:"iprange"`
	WireguardPrivateKeyEncrypted string           `json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string           `json:"wireguard_public_key"`
	WireguardListenPort          int64            `json:"wireguard_listen_port"`
	Peers                        []WireguardPeer1 `json:"peers"`
}

// WireguardPeer1 jsx Schema
type WireguardPeer1 struct {
	PublicKey  string         `json:"public_key"`
	Endpoint   string         `json:"endpoint"`
	AllowedIPs []string       `json:"allowed_iprange"`
	IPRange    schema.IPRange `json:"iprange"`
}
