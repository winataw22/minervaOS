package provision

import (
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
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

// ToProvisionType converts TfgridReservationContainer1 to provision.Container
func (c TfgridReservationContainer1) ToProvisionType() (provision.Container, string, error) {
	container := provision.Container{
		FList:        c.Flist,
		FlistStorage: c.HubURL,
		Env:          c.Environment,
		SecretEnv:    c.SecretEnvironment,
		Entrypoint:   c.Entrypoint,
		Interactive:  c.Interactive,
		Mounts:       make([]provision.Mount, len(c.Volumes)),
	}
	if len(c.NetworkConnection) > 0 {
		container.Network = provision.Network{
			IPs:       []net.IP{c.NetworkConnection[0].Ipaddress},
			NetworkID: pkg.NetID(c.NetworkConnection[0].NetworkID),
		}
	}

	for i, mount := range c.Volumes {
		container.Mounts[i] = provision.Mount{
			VolumeID:   mount.VolumeID,
			Mountpoint: mount.Mountpoint,
		}
	}

	return container, c.NodeID, nil
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
