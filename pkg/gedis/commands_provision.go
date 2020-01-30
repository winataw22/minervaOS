package gedis

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/threefoldtech/zos/pkg/schema"

	types "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/provision"

	"github.com/threefoldtech/zos/pkg"
)

// provisionOrder is used to sort the workload type
// in the right order for provisiond
var provisionOrder = map[provision.ReservationType]int{
	provision.DebugReservation:      0,
	provision.NetworkReservation:    1,
	provision.ZDBReservation:        2,
	provision.VolumeReservation:     3,
	provision.ContainerReservation:  4,
	provision.KubernetesReservation: 5,
}

// Reserve provision.Reserver
func (g *Gedis) Reserve(r *provision.Reservation) (string, error) {
	res := types.TfgridReservation1{
		DataReservation: types.TfgridReservationData1{},
		// CustomerTid:     r.User, //TODO: wrong type.
	}

	w, err := workloadFromRaw(r.Data, r.Type)
	if err != nil {
		return "", err
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

	result, err := Bytes(g.Send("tfgrid.workloads.workload_manager", "reservation_register", Args{
		"reservation": res,
	}))

	if err != nil {
		return "", err
	}

	res = types.TfgridReservation1{}
	if err = json.Unmarshal(result, &res); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", res.ID), nil
}

// Get implements provision.ReservationGetter
func (g *Gedis) Get(id string) (*provision.Reservation, error) {
	result, err := Bytes(g.Send("tfgrid.workloads.workload_manager", "workload_get", Args{
		"gwid": id,
	}))

	if err != nil {
		return nil, err
	}

	var workload types.TfgridReservationWorkload1

	if err = json.Unmarshal(result, &workload); err != nil {
		return nil, err
	}

	return reservationFromSchema(workload)
}

// Poll retrieves reservations from BCDB. from acts like a cursor, first call should use
// 0  to retrieve everything. Next calls should use the last (MAX) ID of the previous poll.
// Note that from is a reservation ID not a workload ID. so user the Reservation.SplitID() method
// to get the reservation part.
func (g *Gedis) Poll(nodeID pkg.Identifier, from uint64) ([]*provision.Reservation, error) {

	result, err := Bytes(g.Send("tfgrid.workloads.workload_manager", "workloads_list", Args{
		"node_id": nodeID.Identity(),
		"cursor":  from,
	}))

	if err != nil {
		return nil, err
	}

	var out struct {
		Workloads []types.TfgridReservationWorkload1 `json:"workloads"`
	}

	if err = json.Unmarshal(result, &out); err != nil {
		return nil, err
	}

	reservations := make([]*provision.Reservation, len(out.Workloads))
	for i, w := range out.Workloads {
		r, err := reservationFromSchema(w)
		if err != nil {
			return nil, err
		}
		reservations[i] = r
	}

	// sorts the primitive in the oder they need to be processed by provisiond
	// network, zdb, volumes, container
	sort.Slice(reservations, func(i int, j int) bool {
		return provisionOrder[reservations[i].Type] < provisionOrder[reservations[j].Type]
	})

	return reservations, nil
}

// Feedback implements provision.Feedbacker
func (g *Gedis) Feedback(id string, r *provision.Result) error {

	var rType types.TfgridReservationResult1CategoryEnum
	switch r.Type {
	case provision.VolumeReservation:
		rType = types.TfgridReservationResult1CategoryVolume
	case provision.ContainerReservation:
		rType = types.TfgridReservationResult1CategoryContainer
	case provision.ZDBReservation:
		rType = types.TfgridReservationResult1CategoryZdb
	case provision.NetworkReservation:
		rType = types.TfgridReservationResult1CategoryNetwork
	}

	var rState types.TfgridReservationResult1StateEnum
	switch r.State {
	case "ok":
		rState = types.TfgridReservationResult1StateOk
	case "error":
		rState = types.TfgridReservationResult1StateError
	}

	result := types.TfgridReservationResult1{
		Category:   rType,
		WorkloadID: id,
		DataJSON:   string(r.Data),
		Signature:  r.Signature,
		State:      rState,
		Message:    r.Error,
		Epoch:      schema.Date{r.Created},
	}

	_, err := g.Send("tfgrid.workloads.workload_manager", "set_workload_result", Args{
		"global_workload_id": id,
		"result":             result,
	})
	return err
}

// Deleted implements provision.Feedbacker
func (g *Gedis) Deleted(id string) error {
	_, err := g.Send("tfgrid.workloads.workload_manager", "workload_deleted", Args{})
	return err
}

// Delete marks a reservation to be deleted
func (g *Gedis) Delete(id string) error {
	_, err := g.Send("tfgrid.workloads.workload_manager", "sign_delete", Args{
		"reservation_id": id,
	})
	return err
}

func reservationFromSchema(w types.TfgridReservationWorkload1) (*provision.Reservation, error) {
	reservation := &provision.Reservation{
		ID:        w.WorkloadID,
		User:      w.User,
		Type:      provision.ReservationType(w.Type.String()),
		Created:   time.Unix(w.Created, 0),
		Duration:  time.Duration(w.Duration) * time.Second,
		Signature: []byte(w.Signature),
		Data:      w.Workload,
		Tag:       provision.Tag{"source": "BCDB"},
	}

	var (
		data interface{}
		err  error
	)

	// convert the workload description from jsx schema to zos types
	switch reservation.Type {
	case provision.ZDBReservation:
		tmp := types.TfgridReservationZdb1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = tmp.ToProvisionType()
		if err != nil {
			return nil, err
		}

	case provision.VolumeReservation:
		tmp := types.TfgridReservationVolume1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = tmp.ToProvisionType()
		if err != nil {
			return nil, err
		}

	case provision.NetworkReservation:
		tmp := types.TfgridReservationNetwork1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, err = tmp.ToProvisionType()
		if err != nil {
			return nil, err
		}

	case provision.ContainerReservation:
		tmp := types.TfgridReservationContainer1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = tmp.ToProvisionType()
		if err != nil {
			return nil, err
		}

	case provision.KubernetesReservation:
		tmp := types.TfgridWorkloadsReservationK8S1{}
		if err := json.Unmarshal(reservation.Data, &tmp); err != nil {
			return nil, err
		}

		data, reservation.NodeID, err = tmp.ToProvisionType()
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

func workloadFromRaw(s json.RawMessage, t provision.ReservationType) (interface{}, error) {
	switch t {
	case provision.ContainerReservation:
		c := provision.Container{}
		err := json.Unmarshal([]byte(s), &c)
		return c, err

	case provision.VolumeReservation:
		v := provision.Volume{}
		err := json.Unmarshal([]byte(s), &v)
		return nil, err

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
			Peers:                        make([]types.WireguardPeer1, len(nr.Peers)),
		}

		for y, peer := range nr.Peers {
			network.NetworkResources[i].Peers[y] = types.WireguardPeer1{
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
		// NodeID:      nodeID,
		Flist:       c.FList,
		HubURL:      c.FlistStorage,
		Environment: c.Env,
		Entrypoint:  c.Entrypoint,
		Interactive: c.Interactive,
		Volumes:     make([]types.TfgridReservationContainerMount1, len(c.Mounts)),
		NetworkConnection: []types.TfgridReservationNetworkConnection1{
			{
				NetworkID: string(c.Network.NetworkID),
				Ipaddress: c.Network.IPs[0],
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
		// WorkloadID:
		// NodeID:
		// ReservationID:
		Size: int64(v.Size),
		// StatsAggregator:
		// FarmerTid:
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
		// WorkloadID:
		// NodeID:
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
		// WorkloadID      int64
		NodeID:        nodeID,
		Size:          k.Size,
		NetworkID:     string(k.NetworkID),
		Ipaddress:     k.IP,
		ClusterSecret: k.ClusterSecret,
		MasterIps:     make([]net.IP, 0, len(k.MasterIPs)),
		SSHKeys:       make([]string, 0, len(k.SSHKeys)),
	}

	for i, ip := range k.MasterIPs {
		k8s.MasterIps[i] = ip
	}

	for i, key := range k.SSHKeys {
		k8s.SSHKeys[i] = key
	}

	return k8s
}
