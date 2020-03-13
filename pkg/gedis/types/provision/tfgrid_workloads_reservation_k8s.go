package provision

import (
	"net"
)

// TfgridWorkloadsReservationK8S1 struct
type TfgridWorkloadsReservationK8S1 struct {
	WorkloadID      int64                               `json:"workload_id"`
	NodeID          string                              `json:"node_id"`
	Size            uint8                               `json:"size"`
	NetworkID       string                              `json:"network_id"`
	Ipaddress       net.IP                              `json:"ipaddress"`
	ClusterSecret   string                              `json:"cluster_secret"`
	MasterIps       []net.IP                            `json:"master_ips"`
	SSHKeys         []string                            `json:"ssh_keys"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	FarmerTid       int64                               `json:"farmer_tid"`
}
