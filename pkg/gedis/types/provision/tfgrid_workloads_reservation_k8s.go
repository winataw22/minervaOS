package provision

import (
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
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

// ToProvisionType converts type to internal provision type
func (k TfgridWorkloadsReservationK8S1) ToProvisionType() (provision.Kubernetes, string, error) {
	k8s := provision.Kubernetes{
		Size:          k.Size,
		NetworkID:     pkg.NetID(k.NetworkID),
		IP:            k.Ipaddress,
		ClusterSecret: k.ClusterSecret,
		MasterIPs:     make([]net.IP, len(k.MasterIps)),
		SSHKeys:       make([]string, len(k.SSHKeys)),
	}

	copy(k8s.MasterIPs, k.MasterIps)
	copy(k8s.SSHKeys, k.SSHKeys)

	return k8s, k.NodeID, nil
}
