package provision

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/threefoldtech/zos/pkg"
	generated "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// ContainerToProvisionType converts TfgridReservationContainer1 to Container
func ContainerToProvisionType(c generated.TfgridReservationContainer1) (Container, string, error) {
	container := Container{
		FList:        c.Flist,
		FlistStorage: c.HubURL,
		Env:          c.Environment,
		SecretEnv:    c.SecretEnvironment,
		Entrypoint:   c.Entrypoint,
		Interactive:  c.Interactive,
		Mounts:       make([]Mount, len(c.Volumes)),
	}
	if len(c.NetworkConnection) > 0 {
		container.Network = Network{
			IPs:       []net.IP{c.NetworkConnection[0].Ipaddress},
			NetworkID: pkg.NetID(c.NetworkConnection[0].NetworkID),
		}
	}

	for i, mount := range c.Volumes {
		container.Mounts[i] = Mount{
			VolumeID:   mount.VolumeID,
			Mountpoint: mount.Mountpoint,
		}
	}

	return container, c.NodeID, nil
}

// VolumeToProvisionType converts TfgridReservationVolume1 to Volume
func VolumeToProvisionType(v generated.TfgridReservationVolume1) (Volume, string, error) {
	volume := Volume{
		Size: uint64(v.Size),
	}
	switch v.Type.String() {
	case "HDD":
		volume.Type = HDDDiskType
	case "SSD":
		volume.Type = SSDDiskType
	default:
		return volume, v.NodeID, fmt.Errorf("disk type %s not supported", v.Type.String())
	}
	return volume, v.NodeID, nil
}

//ZDBToProvisionType converts TfgridReservationZdb1 to ZDB
func ZDBToProvisionType(z generated.TfgridReservationZdb1) (ZDB, string, error) {
	zdb := ZDB{
		Size:     uint64(z.Size),
		Password: z.Password,
		Public:   z.Public,
	}
	switch z.DiskType.String() {
	case "hdd":
		zdb.DiskType = pkg.HDDDevice
	case "ssd":
		zdb.DiskType = pkg.SSDDevice
	default:
		return zdb, z.NodeID, fmt.Errorf("device type %s not supported", z.DiskType.String())
	}

	switch z.Mode.String() {
	case "seq":
		zdb.Mode = pkg.ZDBModeSeq
	case "user":
		zdb.Mode = pkg.ZDBModeUser
	default:
		return zdb, z.NodeID, fmt.Errorf("0-db mode %s not supported", z.Mode.String())
	}

	return zdb, z.NodeID, nil
}

// K8SToProvisionType converts type to internal provision type
func K8SToProvisionType(k generated.TfgridWorkloadsReservationK8S1) (Kubernetes, string, error) {
	k8s := Kubernetes{
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

// NetworkToProvisionType convert TfgridReservationNetwork1 to pkg.Network
func NetworkToProvisionType(n generated.TfgridReservationNetwork1) (pkg.Network, error) {
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
func WireguardToProvisionType(p generated.WireguardPeer1) (pkg.Peer, error) {
	peer := pkg.Peer{
		WGPublicKey: p.PublicKey,
		Endpoint:    p.Endpoint,
		AllowedIPs:  make([]types.IPNet, len(p.AllowedIPs)),
		Subnet:      types.NewIPNetFromSchema(p.IPRange),
	}

	var err error
	for i, ip := range p.AllowedIPs {
		peer.AllowedIPs[i], err = types.ParseIPNet(ip)
		if err != nil {
			return peer, err
		}
	}
	return peer, nil
}

//NetResourceToProvisionType converts TfgridNetworkNetResource1 to pkg.NetResource
func NetResourceToProvisionType(r generated.TfgridNetworkNetResource1) (pkg.NetResource, error) {
	nr := pkg.NetResource{
		NodeID:       r.NodeID,
		Subnet:       types.NewIPNetFromSchema(r.IPRange),
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
func WorkloadToProvisionType(w generated.TfgridReservationWorkload1) (*Reservation, error) {
	reservation := &Reservation{
		ID:        w.WorkloadID,
		User:      w.User,
		Type:      ReservationType(w.Type.String()),
		Created:   w.Created.Time,
		Duration:  time.Duration(w.Duration) * time.Second,
		Signature: []byte(w.Signature),
		Data:      w.Workload,
		ToDelete:  w.ToDelete,
	}

	var (
		data interface{}
		err  error
	)

	// convert the workload description from jsx schema to zos types
	switch reservation.Type {
	case ZDBReservation:
		tmp := generated.TfgridReservationZdb1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = ZDBToProvisionType(tmp)
		if err != nil {
			return nil, err
		}

	case VolumeReservation:
		tmp := generated.TfgridReservationVolume1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = VolumeToProvisionType(tmp)
		if err != nil {
			return nil, err
		}

	case NetworkReservation:
		tmp := generated.TfgridReservationNetwork1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, err = NetworkToProvisionType(tmp)
		if err != nil {
			return nil, err
		}

	case ContainerReservation:
		tmp := generated.TfgridReservationContainer1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = ContainerToProvisionType(tmp)
		if err != nil {
			return nil, err
		}

	case KubernetesReservation:
		tmp := generated.TfgridWorkloadsReservationK8S1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = K8SToProvisionType(tmp)
		if err != nil {
			return nil, err
		}

	}

	reservation.Data, err = json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return reservation, nil
}
