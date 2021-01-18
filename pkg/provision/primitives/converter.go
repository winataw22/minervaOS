package primitives

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
func ContainerToProvisionType(w workloads.Workloader, reservationID string) (Container, string, error) {
	c, ok := w.(*workloads.Container)
	if !ok {
		return Container{}, "", fmt.Errorf("failed to convert container workload, wrong format")
	}

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
		FList:        c.Flist,
		FlistStorage: c.HubUrl,
		Env:          c.Environment,
		SecretEnv:    c.SecretEnvironment,
		Entrypoint:   c.Entrypoint,
		Interactive:  c.Interactive,
		Mounts:       make([]Mount, len(c.Volumes)),
		Logs:         make([]Logs, len(c.Logs)),
		Stats:        make([]stats.Stats, len(c.Stats)),
		Capacity: ContainerCapacity{
			CPU:      uint(c.Capacity.Cpu),
			Memory:   uint64(c.Capacity.Memory),
			DiskType: diskType,
			DiskSize: uint64(c.Capacity.DiskSize),
		},
	}

	if len(c.NetworkConnection) > 0 {
		container.Network = Network{
			IPs:         []net.IP{c.NetworkConnection[0].Ipaddress},
			NetworkID:   pkg.NetID(c.NetworkConnection[0].NetworkId),
			PublicIP6:   c.NetworkConnection[0].PublicIp6,
			YggdrasilIP: c.NetworkConnection[0].YggdrasilIP,
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
			container.Logs[i] = Logs{
				Type: "unknown",
			}
		}

		container.Logs[i] = Logs{
			Type: lg.Type,
			Data: LogsData{
				Stdout:       lg.Data.Stdout,
				Stderr:       lg.Data.Stderr,
				SecretStdout: lg.Data.SecretStdout,
				SecretStderr: lg.Data.SecretStderr,
			},
		}
	}

	unknstats := stats.Stats{
		Type: "unknown",
		Data: stats.Redis{
			Endpoint: "",
		},
	}

	for i, s := range c.Stats {
		// Only support redis for now
		if s.Type != stats.RedisType {
			container.Stats[i] = unknstats
			continue
		}

		data := stats.Redis{}
		err := json.Unmarshal(s.Data, &data)
		if err != nil {
			container.Stats[i] = unknstats
			continue
		}

		container.Stats[i] = stats.Stats{
			Type: s.Type,
			Data: stats.Redis{
				Endpoint: data.Endpoint,
			},
		}
	}

	return container, c.NodeId, nil
}

// VolumeToProvisionType converts TfgridReservationVolume1 to Volume
func VolumeToProvisionType(w workloads.Workloader) (Volume, string, error) {
	v, ok := w.(*workloads.Volume)
	if !ok {
		return Volume{}, "", fmt.Errorf("failed to convert volume workload, wrong format")
	}

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
func ZDBToProvisionType(w workloads.Workloader) (ZDB, string, error) {
	z, ok := w.(*workloads.ZDB)
	if !ok {
		return ZDB{}, "", fmt.Errorf("failed to convert zdb workload, wrong format")
	}

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
func K8SToProvisionType(w workloads.Workloader) (Kubernetes, string, error) {
	k, ok := w.(*workloads.K8S)
	if !ok {
		return Kubernetes{}, "", fmt.Errorf("failed to convert kubernetes workload, wrong format")
	}

	k8s := Kubernetes{
		Size:          uint8(k.Size),
		NetworkID:     pkg.NetID(k.NetworkId),
		IP:            k.Ipaddress,
		ClusterSecret: k.ClusterSecret,
		MasterIPs:     k.MasterIps,
		SSHKeys:       k.SshKeys,
		PublicIP:      k.PublicIP,
	}

	return k8s, k.NodeId, nil
}

// PublicIPToProvisionType converts type to internal provision type
func PublicIPToProvisionType(w workloads.Workloader) (PublicIP, string, error) {
	p, ok := w.(*workloads.PublicIP)
	if !ok {
		return PublicIP{}, "", fmt.Errorf("failed to convert kubernetes workload, wrong format")
	}

	publicIP := PublicIP{
		IP: p.IPaddress.IPNet,
	}

	return publicIP, p.NodeId, nil
}

// NetworkResourceToProvisionType converts type to internal provision type
func NetworkResourceToProvisionType(w workloads.Workloader) (pkg.NetResource, error) {
	n, ok := w.(*workloads.NetworkResource)
	if !ok {
		return pkg.NetResource{}, fmt.Errorf("failed to convert kubernetes workload, wrong format")
	}

	nr := pkg.NetResource{
		Name:           n.Name,
		NetID:          pkg.NetID(n.Name),
		NetworkIPRange: types.NewIPNetFromSchema(n.NetworkIprange),

		NodeID:       n.GetNodeID(),
		Subnet:       types.NewIPNetFromSchema(n.Iprange),
		WGPrivateKey: n.WireguardPrivateKeyEncrypted,
		WGPublicKey:  n.WireguardPublicKey,
		WGListenPort: uint16(n.WireguardListenPort),
		Peers:        make([]pkg.Peer, len(n.Peers)),
	}

	for i, peer := range n.Peers {
		p, err := WireguardToProvisionType(peer)
		if err != nil {
			return nr, err
		}
		nr.Peers[i] = p
	}

	return nr, nil
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

// WorkloadToProvisionType converts from the explorer type to the internal provision.Reservation
func WorkloadToProvisionType(w workloads.Workloader) (*provision.Reservation, error) {
	nextAction := w.GetNextAction()
	reservation := &provision.Reservation{
		ID:        fmt.Sprintf("%d-%d", w.GetID(), w.WorkloadID()),
		User:      fmt.Sprintf("%d", w.GetCustomerTid()),
		Type:      provision.ReservationType(w.GetWorkloadType().String()),
		Created:   w.GetEpoch().Time,
		Duration:  math.MaxInt64, //ensure we never decomission based on expiration time. Since the capacity pool introduction this is not needed anymore
		Signature: []byte(w.GetCustomerSignature()),
		ToDelete:  nextAction == workloads.NextActionDelete || nextAction == workloads.NextActionDeleted,
		Reference: w.GetReference(),
		Result:    resultFromSchemaType(w.GetResult()),
		Version:   w.GetVersion(),
	}

	var (
		data interface{}
		err  error
	)

	switch w.GetWorkloadType() {
	case workloads.WorkloadTypeZDB:
		data, reservation.NodeID, err = ZDBToProvisionType(w)
		if err != nil {
			return nil, err
		}
	case workloads.WorkloadTypeVolume:
		data, reservation.NodeID, err = VolumeToProvisionType(w)
		if err != nil {
			return nil, err
		}
	case workloads.WorkloadTypeNetworkResource:
		data, err = NetworkResourceToProvisionType(w)
		if err != nil {
			return nil, err
		}
	case workloads.WorkloadTypeContainer:
		reservationID := strings.Split(reservation.ID, "-")[0]
		data, reservation.NodeID, err = ContainerToProvisionType(w, reservationID)
		if err != nil {
			return nil, err
		}
	case workloads.WorkloadTypeKubernetes:
		data, reservation.NodeID, err = K8SToProvisionType(w)
		if err != nil {
			return nil, err
		}
	case workloads.WorkloadTypePublicIP:
		data, reservation.NodeID, err = PublicIPToProvisionType(w)
		if err != nil {
			return nil, err
		}
	default:
		log.Error().Str("type", w.GetWorkloadType().String()).Msg("unsupported reservation type")
		return reservation, nil
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
	case NetworkReservation, NetworkResourceReservation:
		rType = workloads.WorkloadTypeNetwork
	case KubernetesReservation:
		rType = workloads.WorkloadTypeKubernetes
	case PublicIPReservation:
		rType = workloads.WorkloadTypePublicIP
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

func resultFromSchemaType(r workloads.Result) provision.Result {

	result := provision.Result{
		Type:      provision.ReservationType(r.Category.String()),
		Created:   r.Epoch.Time,
		Data:      r.DataJson,
		Error:     r.Message,
		ID:        r.WorkloadId,
		State:     provision.ResultState(r.State),
		Signature: r.Signature,
	}

	return result
}
