package gedis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/threefoldtech/zos/pkg/schema"

	types "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/provision"

	"github.com/threefoldtech/zos/pkg"
)

// Reserve provision.Reserver
func (g *Gedis) Reserve(r *provision.Reservation, nodeID pkg.Identifier) (string, error) {
	res := types.TfgridReservation1{
		DataReservation: types.TfgridReservationData1{},
		// CustomerTid:     r.User, //TODO: wrong type.
	}

	w, err := workloadFromRaw(r.Data, r.Type)
	if err != nil {
		return "", err
	}
	nID := nodeID.Identity()

	switch r.Type {
	case provision.ContainerReservation:
		res.DataReservation.Containers = []types.TfgridReservationContainer1{
			containerReservation(w, nID),
		}
	case provision.VolumeReservation:
		res.DataReservation.Volumes = []types.TfgridReservationVolume1{
			volumeReservation(w, nID),
		}
	case provision.ZDBReservation:
		res.DataReservation.Zdbs = []types.TfgridReservationZdb1{
			zdbReservation(w, nID),
		}
	case provision.NetworkReservation:
		res.DataReservation.Networks = []types.TfgridReservationNetwork1{
			networkReservation(w),
		}
	}

	result, err := Bytes(g.Send("workload_manager", "reservation_register", Args{
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
	result, err := Bytes(g.Send("workload_manager", "workload_get", Args{
		"gwid": id,
	}))

	if err != nil {
		return nil, err
	}

	var workload types.TfgridReservationWorkload1

	if err = json.Unmarshal(result, &workload); err != nil {
		return nil, err
	}

	return reservationFromSchema(workload), nil
}

// Poll implements provision.ReservationPoller
func (g *Gedis) Poll(nodeID pkg.Identifier, all bool, since time.Time) ([]*provision.Reservation, error) {

	epoch := since.Unix()
	// all means sends all reservation so we ask since the beginning of (unix) time
	if all {
		epoch = 0
	}
	result, err := Bytes(g.Send("workload_manager", "workloads_list", Args{
		"node_id": nodeID.Identity(),
		"epoch":   epoch,
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
		reservations[i] = reservationFromSchema(w)
	}

	return reservations, nil
}

// Feedback implements provision.Feedbacker
func (g *Gedis) Feedback(id string, r *provision.Result) error {
	rID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
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
		WorkloadID: rID,
		DataJSON:   string(r.Data),
		Signature:  r.Signature,
		State:      rState,
		Message:    r.Error,
		Epoch:      schema.Date{r.Created},
	}

	_, err = g.Send("workload_manager", "set_workload_result", Args{
		"reservation_id": rID,
		"result":         result,
	})
	return err
}

// Deleted implements provision.Feedbacker
func (g *Gedis) Deleted(id string) error { return nil }

func reservationFromSchema(w types.TfgridReservationWorkload1) *provision.Reservation {
	return &provision.Reservation{
		ID:        w.WorkloadID,
		User:      w.User,
		Type:      provision.ReservationType(w.Type.String()),
		Created:   time.Unix(w.Created, 0),
		Duration:  time.Duration(w.Duration) * time.Second,
		Signature: []byte(w.Signature),
		Data:      w.Workload,
	}
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
				NetworkID: string(c.Network.NetwokID),
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
