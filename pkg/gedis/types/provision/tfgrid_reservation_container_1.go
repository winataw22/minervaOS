package provision

import (
	"net"
)

//TfgridReservationContainer1 jsx schema
type TfgridReservationContainer1 struct {
	WorkloadID        int64                                 `json:"workload_id"`
	NodeID            string                                `json:"node_id"`
	Flist             string                                `json:"flist"`
	HubURL            string                                `json:"hub_url"`
	Environment       map[string]string                     `json:"environment"`
	SecretEnvironment map[string]string                     `json:"secret_environment"`
	Entrypoint        string                                `json:"entrypoint"`
	Interactive       bool                                  `json:"interactive"`
	Volumes           []TfgridReservationContainerMount1    `json:"volumes"`
	NetworkConnection []TfgridReservationNetworkConnection1 `json:"network_connection"`
	StatsAggregator   []TfgridReservationStatsaggregator1   `json:"stats_aggregator"`
}

//TfgridReservationContainerMount1 jsx schema
type TfgridReservationContainerMount1 struct {
	VolumeID   string `json:"volume_id"`
	Mountpoint string `json:"mountpoint"`
}

//TfgridReservationNetworkConnection1 jsx schema
type TfgridReservationNetworkConnection1 struct {
	NetworkID string `json:"network_id"`
	Ipaddress net.IP `json:"ipaddress"`
	PublicIP6 bool   `json:"public_ip6"`
}
