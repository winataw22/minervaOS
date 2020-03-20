package gedis

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/threefoldtech/zos/pkg"
	types "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/schema"
)

// ReservationToSchemaType creates a TfgridReservation1 from zos provision types
func ReservationToSchemaType(r *provision.Reservation) (types.TfgridReservation1, error) {
	var res types.TfgridReservation1

	w, err := workloadFromRaw(r.Data, r.Type)
	if err != nil {
		return res, err
	}

	switch r.Type {
	case provision.ContainerReservation:
		res.DataReservation.Containers = []types.TfgridReservationContainer1{
			containerReservation(w, r.NodeID),
		}
	case provision.VolumeReservation:
		res.DataReservation.Volumes = []types.TfgridReservationVolume1{
			volumeReservation(w, r.NodeID),
		}
	case provision.ZDBReservation:
		res.DataReservation.Zdbs = []types.TfgridReservationZdb1{
			zdbReservation(w, r.NodeID),
		}
	case provision.NetworkReservation:
		res.DataReservation.Networks = []types.TfgridReservationNetwork1{
			networkReservation(w),
		}
	case provision.KubernetesReservation:
		res.DataReservation.Kubernetes = []types.TfgridWorkloadsReservationK8S1{
			k8sReservation(w, r.NodeID),
		}
	}

	res.Epoch = schema.Date{Time: r.Created}
	res.DataReservation.ExpirationReservation = schema.Date{Time: r.Created.Add(r.Duration)}
	res.DataReservation.ExpirationProvisioning = schema.Date{Time: r.Created.Add(2 * time.Minute)}

	return res, nil
}

func workloadFromRaw(s json.RawMessage, t provision.ReservationType) (interface{}, error) {
	switch t {
	case provision.ContainerReservation:
		c := provision.Container{}
		err := json.Unmarshal([]byte(s), &c)
		return c, err

	case provision.VolumeReservation:
		v := provision.Volume{}
		err := json.Unmarshal([]byte(s), &v)
		return v, err

	case provision.NetworkReservation:
		n := pkg.Network{}
		err := json.Unmarshal([]byte(s), &n)
		return n, err

	case provision.ZDBReservation:
		z := provision.ZDB{}
		err := json.Unmarshal([]byte(s), &z)
		return z, err

	case provision.KubernetesReservation:
		k := provision.Kubernetes{}
		err := json.Unmarshal([]byte(s), &k)
		return k, err
	}

	return nil, fmt.Errorf("unsupported reservation type %v", t)
}

func networkReservation(i interface{}) types.TfgridReservationNetwork1 {
	n := i.(pkg.Network)
	network := types.TfgridReservationNetwork1{
		Name:             n.Name,
		Iprange:          n.IPRange.ToSchema(),
		WorkloadID:       1,
		NetworkResources: make([]types.TfgridNetworkNetResource1, len(n.NetResources)),
	}

	for i, nr := range n.NetResources {
		network.NetworkResources[i] = types.TfgridNetworkNetResource1{
			NodeID:                       nr.NodeID,
			IPRange:                      nr.Subnet.ToSchema(),
			WireguardPrivateKeyEncrypted: nr.WGPrivateKey,
			WireguardPublicKey:           nr.WGPublicKey,
			WireguardListenPort:          int64(nr.WGListenPort),
			Peers:                        make([]types.WireguardPeer1, len(nr.Peers)),
		}

		for y, peer := range nr.Peers {
			network.NetworkResources[i].Peers[y] = types.WireguardPeer1{
				IPRange:    peer.Subnet.ToSchema(),
				Endpoint:   peer.Endpoint,
				PublicKey:  peer.WGPublicKey,
				AllowedIPs: make([]string, len(peer.AllowedIPs)),
			}

			for z, ip := range peer.AllowedIPs {
				network.NetworkResources[i].Peers[y].AllowedIPs[z] = ip.String()
			}
		}
	}
	return network
}

func containerReservation(i interface{}, nodeID string) types.TfgridReservationContainer1 {
	c := i.(provision.Container)
	container := types.TfgridReservationContainer1{
		NodeID:            nodeID,
		WorkloadID:        1,
		Flist:             c.FList,
		HubURL:            c.FlistStorage,
		Environment:       c.Env,
		SecretEnvironment: c.SecretEnv,
		Entrypoint:        c.Entrypoint,
		Interactive:       c.Interactive,
		Volumes:           make([]types.TfgridReservationContainerMount1, len(c.Mounts)),
		NetworkConnection: []types.TfgridReservationNetworkConnection1{
			{
				NetworkID: string(c.Network.NetworkID),
				Ipaddress: c.Network.IPs[0],
				PublicIP6: c.Network.PublicIP6,
			},
		},
		// StatsAggregator:   c.StatsAggregator,
		// FarmerTid:         c.FarmerTid,
	}

	for i, v := range c.Mounts {
		container.Volumes[i] = types.TfgridReservationContainerMount1{
			VolumeID:   v.VolumeID,
			Mountpoint: v.Mountpoint,
		}
	}
	return container
}

func volumeReservation(i interface{}, nodeID string) types.TfgridReservationVolume1 {
	v := i.(provision.Volume)

	volume := types.TfgridReservationVolume1{
		NodeID:     nodeID,
		WorkloadID: 1,
		Size:       int64(v.Size),
	}
	if v.Type == provision.HDDDiskType {
		volume.Type = types.TfgridReservationVolume1TypeHDD
	} else if v.Type == provision.SSDDiskType {
		volume.Type = types.TfgridReservationVolume1TypeSSD
	}

	return volume
}

func zdbReservation(i interface{}, nodeID string) types.TfgridReservationZdb1 {
	z := i.(provision.ZDB)

	zdb := types.TfgridReservationZdb1{
		WorkloadID: 1,
		NodeID:     nodeID,
		// ReservationID:
		Size:     int64(z.Size),
		Password: z.Password,
		Public:   z.Public,
		// StatsAggregator:
		// FarmerTid:
	}
	if z.DiskType == pkg.SSDDevice {
		zdb.DiskType = types.TfgridReservationZdb1DiskTypeHdd
	} else if z.DiskType == pkg.HDDDevice {
		zdb.DiskType = types.TfgridReservationZdb1DiskTypeSsd
	}

	if z.Mode == pkg.ZDBModeUser {
		zdb.Mode = types.TfgridReservationZdb1ModeUser
	} else if z.Mode == pkg.ZDBModeSeq {
		zdb.Mode = types.TfgridReservationZdb1ModeSeq
	}

	return zdb
}

func k8sReservation(i interface{}, nodeID string) types.TfgridWorkloadsReservationK8S1 {
	k := i.(provision.Kubernetes)

	k8s := types.TfgridWorkloadsReservationK8S1{
		WorkloadID:    1,
		NodeID:        nodeID,
		Size:          k.Size,
		NetworkID:     string(k.NetworkID),
		Ipaddress:     k.IP,
		ClusterSecret: k.ClusterSecret,
		MasterIps:     make([]net.IP, len(k.MasterIPs)),
		SSHKeys:       make([]string, len(k.SSHKeys)),
	}

	copy(k8s.MasterIps, k.MasterIPs)
	copy(k8s.SSHKeys, k.SSHKeys)

	return k8s
}
