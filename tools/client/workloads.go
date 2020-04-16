package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
	wrklds "github.com/threefoldtech/zos/tools/explorer/pkg/workloads"
)

type httpWorkloads struct {
	*httpClient
}

func (w *httpWorkloads) Create(reservation workloads.Reservation) (resp wrklds.ReservationCreateResponse, err error) {
	err = w.post(w.url("reservations"), reservation, &resp, http.StatusCreated)
	return
}

func (w *httpWorkloads) List(nextAction *workloads.NextActionEnum, customerTid int64, page *Pager) (reservation []workloads.Reservation, err error) {
	query := url.Values{}
	if nextAction != nil {
		query.Set("next_action", fmt.Sprintf("%d", nextAction))
	}
	if customerTid != 0 {
		query.Set("customer_tid", fmt.Sprint(customerTid))
	}
	page.apply(query)

	err = w.get(w.url("reservations"), query, &reservation, http.StatusOK)
	return
}

func (w *httpWorkloads) Get(id schema.ID) (reservation workloads.Reservation, err error) {
	err = w.get(w.url("reservations", fmt.Sprint(id)), nil, &reservation, http.StatusOK)
	return
}

func (w *httpWorkloads) SignProvision(id schema.ID, user schema.ID, signature string) error {
	return w.post(
		w.url("reservations", fmt.Sprint(id), "sign", "provision"),
		workloads.SigningSignature{
			Tid:       int64(user),
			Signature: signature,
		},
		nil,
		http.StatusCreated,
	)
}

func (w *httpWorkloads) SignDelete(id schema.ID, user schema.ID, signature string) error {
	return w.post(
		w.url("reservations", fmt.Sprint(id), "sign", "delete"),
		workloads.SigningSignature{
			Tid:       int64(user),
			Signature: signature,
		},
		nil,
		http.StatusCreated,
	)
}

type intermediateWL struct {
	workloads.ReservationWorkload
	Content json.RawMessage `json:"content"`
}

func (wl *intermediateWL) Workload() (result workloads.ReservationWorkload, err error) {
	result = wl.ReservationWorkload
	switch wl.Type {
	case workloads.WorkloadTypeContainer:
		var o workloads.Container
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.WorkloadTypeKubernetes:
		var o workloads.K8S
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.WorkloadTypeNetwork:
		var o workloads.Network
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.WorkloadTypeVolume:
		var o workloads.Volume
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.WorkloadTypeZDB:
		var o workloads.ZDB
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	default:
		return result, fmt.Errorf("unknown workload type")
	}

	return
}

func (w *httpWorkloads) Workloads(nodeID string, from uint64) ([]workloads.ReservationWorkload, error) {
	query := url.Values{}
	query.Set("from", fmt.Sprint(from))

	var list []intermediateWL

	err := w.get(
		w.url("reservations", "workloads", nodeID),
		query,
		&list,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	results := make([]workloads.ReservationWorkload, 0, len(list))
	for _, i := range list {
		wl, err := i.Workload()
		if err != nil {
			return nil, err
		}
		results = append(results, wl)
	}
	return results, err
}

func (w *httpWorkloads) WorkloadGet(gwid string) (result workloads.ReservationWorkload, err error) {
	var output intermediateWL
	err = w.get(w.url("reservations", "workloads", gwid), nil, &output, http.StatusOK)
	if err != nil {
		return
	}

	return output.Workload()
}

func (w *httpWorkloads) WorkloadPutResult(nodeID, gwid string, result workloads.Result) error {
	return w.put(w.url("reservations", "workloads", gwid, nodeID), result, nil, http.StatusCreated)
}

func (w *httpWorkloads) WorkloadPutDeleted(nodeID, gwid string) error {
	return w.delete(w.url("reservations", "workloads", gwid, nodeID), nil, nil, http.StatusOK)
}
