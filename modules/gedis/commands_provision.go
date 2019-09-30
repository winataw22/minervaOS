package gedis

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/threefoldtech/zosv2/modules/schema"

	types "github.com/threefoldtech/zosv2/modules/gedis/types/provision"
	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/threefoldtech/zosv2/modules"
)

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
func (g *Gedis) Poll(nodeID modules.Identifier, all bool, since time.Time) ([]*provision.Reservation, error) {

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

	b, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = g.Send("workload_manager", "set_workload_result", Args{
		"reservation_id": rID,
		"result":         b,
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
