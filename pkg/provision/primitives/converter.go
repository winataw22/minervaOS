package primitives

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/provision"
)

// ErrUnsupportedWorkload is return when a workload of a type not supported by
// provisiond is received from the explorer
var ErrUnsupportedWorkload = errors.New("workload type not supported")

// ContainerToProvisionType converts TfgridReservationContainer1 to Container
func ContainerToProvisionType(c workloads.Container, reservationID string) (Container, string, error) {
	var diskType pkg.DeviceType
	switch strings.ToLower(c.Capacity.DiskType.String()) {
	case "hdd":
		diskType = pkg.HDDDevice
	case "ssd":
		diskType = pkg.SSDDevice
	default:
		return Container{}, "", fmt.Errorf("unknown disk type: %s", c.Capacity.DiskType.String())
	}

	container := Container{
		FList:           c.Flist,
		FlistStorage:    c.HubUrl,
		Env:             c.Environment,
		SecretEnv:       c.SecretEnvironment,
		Entrypoint:      c.Entrypoint,
		Interactive:     c.Interactive,
		Mounts:          make([]Mount, len(c.Volumes)),
		Logs:            make([]logger.Logs, len(c.Logs)),
		StatsAggregator: make([]stats.Aggregator, len(c.StatsAggregator)),
		Capacity: ContainerCapacity{
			CPU:      uint(c.Capacity.Cpu),
			Memory:   uint64(c.Capacity.Memory),
			DiskType: diskType,
			DiskSize: uint64(c.Capacity.DiskSize),
		},
	}

	if len(c.NetworkConnection) > 0 {
		container.Network = Network{
			IPs:       []net.IP{c.NetworkConnection[0].Ipaddress},
			NetworkID: pkg.NetID(c.NetworkConnection[0].NetworkId),
			PublicIP6: c.NetworkConnection[0].PublicIp6,
		}
	}

	for i, mount := range c.Volumes {
		if strings.HasPrefix(mount.VolumeId, "-") {
			mount.VolumeId = reservationID + mount.VolumeId
		}
		container.Mounts[i] = Mount{
			VolumeID:   mount.VolumeId,
			Mountpoint: mount.Mountpoint,
		}
	}

	for i, lg := range c.Logs {
		// Only support redis for now
		if lg.Type != logger.RedisType {
			container.Logs[i] = logger.Logs{
				Type: "unknown",
				Data: logger.LogsRedis{
					Stdout: "",
					Stderr: "",
				},
			}
		}

		container.Logs[i] = logger.Logs{
			Type: lg.Type,
			Data: logger.LogsRedis{
				Stdout: lg.Data.Stdout,
				Stderr: lg.Data.Stderr,
			},
		}
	}

	for i, s := range c.StatsAggregator {
		// Only support redis for now
		if s.Type != stats.RedisType {
			container.StatsAggregator[i] = stats.Aggregator{
				Type: "unknown",
				Data: stats.Redis{
					Endpoint: "",
				},
			}
		}

		container.StatsAggregator[i] = stats.Aggregator{
			Type: s.Type,
			Data: stats.Redis{
				Endpoint: s.Data.Endpoint,
			},
		}
	}

	return container, c.NodeId, nil
}

// VolumeToProvisionType converts TfgridReservationVolume1 to Volume
func VolumeToProvisionType(v workloads.Volume) (Volume, string, error) {
	volume := Volume{
		Size: uint64(v.Size),
	}
	switch strings.ToLower(v.Type.String()) {
	case "hdd":
		volume.Type = pkg.HDDDevice
	case "ssd":
		volume.Type = pkg.SSDDevice
	default:
		return volume, v.NodeId, fmt.Errorf("disk type %s not supported", v.Type.String())
	}
	return volume, v.NodeId, nil
}

//ZDBToProvisionType converts TfgridReservationZdb1 to ZDB
func ZDBToProvisionType(z workloads.ZDB) (ZDB, string, error) {
	zdb := ZDB{
		Size:     uint64(z.Size),
		Password: z.Password,
		Public:   z.Public,
	}
	switch strings.ToLower(z.DiskType.String()) {
	case "hdd":
		zdb.DiskType = pkg.HDDDevice
	case "ssd":
		zdb.DiskType = pkg.SSDDevice
	default:
		return zdb, z.NodeId, fmt.Errorf("device type %s not supported", z.DiskType.String())
	}

	switch z.Mode.String() {
	case "seq":
		zdb.Mode = pkg.ZDBModeSeq
	case "user":
		zdb.Mode = pkg.ZDBModeUser
	default:
		return zdb, z.NodeId, fmt.Errorf("0-db mode %s not supported", z.Mode.String())
	}

	return zdb, z.NodeId, nil
}

// K8SToProvisionType converts type to internal provision type
func K8SToProvisionType(k workloads.K8S) (Kubernetes, string, error) {
	k8s := Kubernetes{
		Size:          uint8(k.Size),
		NetworkID:     pkg.NetID(k.NetworkId),
		IP:            k.Ipaddress,
		ClusterSecret: k.ClusterSecret,
		MasterIPs:     k.MasterIps,
		SSHKeys:       k.SshKeys,
	}

	return k8s, k.NodeId, nil
}

// NetworkToProvisionType convert TfgridReservationNetwork1 to pkg.Network
func NetworkToProvisionType(n workloads.Network) (pkg.Network, error) {
	network := pkg.Network{
		Name:         n.Name,
		NetID:        pkg.NetID(n.Name),
		IPRange:      types.NewIPNetFromSchema(n.Iprange),
		NetResources: make([]pkg.NetResource, len(n.NetworkResources)),
	}

	var err error
	for i, nr := range n.NetworkResources {
		network.NetResources[i], err = NetResourceToProvisionType(nr)
		if err != nil {
			return network, err
		}
	}
	return network, nil
}

//WireguardToProvisionType converts WireguardPeer1 to pkg.Peer
func WireguardToProvisionType(p workloads.WireguardPeer) (pkg.Peer, error) {
	peer := pkg.Peer{
		WGPublicKey: p.PublicKey,
		Endpoint:    p.Endpoint,
		AllowedIPs:  make([]types.IPNet, len(p.AllowedIprange)),
		Subnet:      types.NewIPNetFromSchema(p.Iprange),
	}

	for i, ip := range p.AllowedIprange {
		peer.AllowedIPs[i] = types.IPNet{IPNet: ip.IPNet}
	}
	return peer, nil
}

//NetResourceToProvisionType converts TfgridNetworkNetResource1 to pkg.NetResource
func NetResourceToProvisionType(r workloads.NetworkNetResource) (pkg.NetResource, error) {
	nr := pkg.NetResource{
		NodeID:       r.NodeId,
		Subnet:       types.NewIPNetFromSchema(r.Iprange),
		WGPrivateKey: r.WireguardPrivateKeyEncrypted,
		WGPublicKey:  r.WireguardPublicKey,
		WGListenPort: uint16(r.WireguardListenPort),
		Peers:        make([]pkg.Peer, len(r.Peers)),
	}

	for i, peer := range r.Peers {
		p, err := WireguardToProvisionType(peer)
		if err != nil {
			return nr, err
		}
		nr.Peers[i] = p
	}

	return nr, nil
}

// WorkloadToProvisionType TfgridReservationWorkload1 to provision.Reservation
func WorkloadToProvisionType(w workloads.ReservationWorkload) (*provision.Reservation, error) {
	reservation := &provision.Reservation{
		ID:        w.WorkloadId,
		User:      w.User,
		Type:      provision.ReservationType(w.Type.String()),
		Created:   w.Created.Time,
		Duration:  time.Duration(w.Duration) * time.Second,
		Signature: []byte(w.Signature),
		// Data:      w.Content,
		ToDelete: w.ToDelete,
	}

	reservationID := strings.Split(w.WorkloadId, "-")[0]

	var (
		data interface{}
		err  error
	)

	switch tmp := w.Content.(type) {
	case workloads.ZDB:
		data, reservation.NodeID, err = ZDBToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.Volume:
		data, reservation.NodeID, err = VolumeToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.Network:
		data, err = NetworkToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.Container:

		data, reservation.NodeID, err = ContainerToProvisionType(tmp, reservationID)
		if err != nil {
			return nil, err
		}
	case workloads.K8S:
		data, reservation.NodeID, err = K8SToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w (%s) (%T)", ErrUnsupportedWorkload, w.Type.String(), tmp)
	}

	reservation.Data, err = json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return reservation, nil
}

// ResultToSchemaType converts result to schema type
func ResultToSchemaType(r provision.Result) (*workloads.Result, error) {

	var rType workloads.WorkloadTypeEnum
	switch r.Type {
	case VolumeReservation:
		rType = workloads.WorkloadTypeVolume
	case ContainerReservation:
		rType = workloads.WorkloadTypeContainer
	case ZDBReservation:
		rType = workloads.WorkloadTypeZDB
	case NetworkReservation:
		rType = workloads.WorkloadTypeNetwork
	case KubernetesReservation:
		rType = workloads.WorkloadTypeKubernetes
	default:
		return nil, fmt.Errorf("unknown reservation type: %s", r.Type)
	}

	result := workloads.Result{
		Category:   rType,
		WorkloadId: r.ID,
		DataJson:   r.Data,
		Signature:  r.Signature,
		State:      workloads.ResultStateEnum(r.State),
		Message:    r.Error,
		Epoch:      schema.Date{Time: r.Created},
	}

	return &result, nil
}
