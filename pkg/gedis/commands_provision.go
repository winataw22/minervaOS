package gedis

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/threefoldtech/zos/pkg/schema"

	dtypes "github.com/threefoldtech/zos/pkg/gedis/types/directory"
	ptypes "github.com/threefoldtech/zos/pkg/gedis/types/provision"
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
	res, err := ReservationToSchemaType(r)
	if err != nil {
		return "", err
	}

	result, err := Bytes(g.Send("tfgrid.workloads.workload_manager", "reservation_register", Args{
		"reservation": res,
	}))

	if err != nil {
		return "", err
	}

	res = ptypes.TfgridReservation1{}
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

	var workload ptypes.TfgridReservationWorkload1

	if err = json.Unmarshal(result, &workload); err != nil {
		return nil, err
	}

	return provision.WorkloadToProvisionType(workload)
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
		return nil, provision.NewErrTemporary(err)
	}

	var out struct {
		Workloads []ptypes.TfgridReservationWorkload1 `json:"workloads"`
	}

	if err = json.Unmarshal(result, &out); err != nil {
		return nil, err
	}

	reservations := make([]*provision.Reservation, len(out.Workloads))
	for i, w := range out.Workloads {
		r, err := provision.WorkloadToProvisionType(w)
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
func (g *Gedis) Feedback(nodeID string, r *provision.Result) error {

	var rType ptypes.TfgridReservationResult1CategoryEnum
	switch r.Type {
	case provision.VolumeReservation:
		rType = ptypes.TfgridReservationResult1CategoryVolume
	case provision.ContainerReservation:
		rType = ptypes.TfgridReservationResult1CategoryContainer
	case provision.ZDBReservation:
		rType = ptypes.TfgridReservationResult1CategoryZdb
	case provision.NetworkReservation:
		rType = ptypes.TfgridReservationResult1CategoryNetwork
	}

	result := ptypes.TfgridReservationResult1{
		Category:   rType,
		WorkloadID: r.ID,
		DataJSON:   r.Data,
		Signature:  r.Signature,
		State:      ptypes.TfgridReservationResult1StateEnum(r.State),
		Message:    r.Error,
		Epoch:      schema.Date{r.Created},
	}

	_, err := g.Send("tfgrid.workloads.workload_manager", "set_workload_result", Args{
		"global_workload_id": r.ID,
		"result":             result,
	})
	return err
}

//UpdateReservedResources sends current reserved resources
func (g *Gedis) UpdateReservedResources(nodeID string, c provision.Counters) error {
	r := dtypes.TfgridNodeResourceAmount1{
		Cru: c.CRU.Current(),
		Mru: c.MRU.Current(),
		Hru: c.HRU.Current(),
		Sru: c.SRU.Current(),
	}
	_, err := g.Send("tfgrid.directory.nodes", "update_reserved_capacity", Args{
		"node_id":   nodeID,
		"resources": r,
	})
	return err
}

// Deleted implements provision.Feedbacker
func (g *Gedis) Deleted(nodeID, id string) error {
	_, err := g.Send("tfgrid.workloads.workload_manager", "workload_deleted", Args{"workload_id": id})
	return err
}

// Delete marks a reservation to be deleted
func (g *Gedis) Delete(id string) error {
	_, err := g.Send("tfgrid.workloads.workload_manager", "sign_delete", Args{
		"reservation_id": id,
	})
	return err
}
