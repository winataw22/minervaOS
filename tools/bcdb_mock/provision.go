package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision"
)

func reserve(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]

	_, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node %s not found", nodeID), http.StatusNotFound)
		return
	}

	defer r.Body.Close()
	res := &provision.Reservation{}
	if err := json.NewDecoder(r.Body).Decode(res); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	provStore.Lock()
	defer provStore.Unlock()

	res.ID = fmt.Sprintf("r-%d", len(provStore.Reservations))
	provStore.Reservations = append(provStore.Reservations, &reservation{
		Reservation: res,
		NodeID:      nodeID,
	})
	w.Header().Set("Location", "/reservations/"+res.ID)
	w.WriteHeader(http.StatusCreated)
}

func pollReservations(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	var since time.Time
	s := r.URL.Query().Get("since")
	if s == "" {
		// if since is not specificed, send all reservation since last hour
		since = time.Now().Add(-time.Hour)
	} else {
		timestamp, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			http.Error(w, "since query argument format not valid", http.StatusBadRequest)
			return
		}
		since = time.Unix(timestamp, 0)
	}

	_, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node %s not found", nodeID), http.StatusNotFound)
		return
	}

	all, err := strconv.ParseBool(r.URL.Query().Get("all"))
	if err != nil {
		all = false
	}

	output := []*provision.Reservation{}
	if all {
		// just get all reservation for this nodeID
		output = getRes(nodeID, all, since)
	} else {
		// otherwise start long polling
		timeout := time.Now().Add(time.Second * 20)
		for {
			output = getRes(nodeID, all, since)
			if len(output) > 0 {
				break
			}

			if time.Now().After(timeout) {
				break
			}
			time.Sleep(time.Second)
		}
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(output); err != nil {
		log.Printf("error encoding empty reservation slice: %v", err)
	}
}

func getRes(nodeID string, all bool, since time.Time) []*provision.Reservation {
	output := []*provision.Reservation{}

	provStore.Lock()
	defer provStore.Unlock()

	for _, r := range provStore.Reservations {
		// skip reservation aimed at another node
		if r.NodeID != nodeID {
			continue
		}

		if all ||
			(!r.Reservation.Expired() && since.Before(r.Reservation.Created)) ||
			(r.Reservation.ToDelete && !r.Deleted) {
			output = append(output, r.Reservation)
		}
	}

	return output
}

func getReservation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	provStore.Lock()
	defer provStore.Unlock()

	w.Header().Add("content-type", "application/json")

	obj := struct {
		Reservation *provision.Reservation `json:"reservation"`
		Result      *provision.Result      `json:"result"`
	}{}

	for _, r := range provStore.Reservations {
		if r.Reservation.ID == id {
			w.WriteHeader(http.StatusOK)
			obj.Reservation = r.Reservation
			obj.Result = r.Result
			if err := json.NewEncoder(w).Encode(obj); err != nil {
				log.Printf("error during json encoding of reservation: %v", err)
			}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func reservationResult(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	provStore.Lock()

	var rsvt *reservation
	for _, rsvt = range provStore.Reservations {
		if rsvt.Reservation.ID == id {
			break
		}
	}
	provStore.Unlock()

	if r == nil {
		http.Error(w, fmt.Sprintf("reservation %s not found", id), http.StatusNotFound)
		return
	}

	w.Header().Add("content-type", "application/json")

	defer r.Body.Close()
	result := &provision.Result{}
	if err := json.NewDecoder(r.Body).Decode(result); err != nil {
		log.Printf("failed to decode reservation result: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rsvt.Result = result

	w.WriteHeader(http.StatusOK)
}

func reservationDeleted(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	provStore.Lock()
	defer provStore.Unlock()

	var rsvt *reservation
	for _, rsvt = range provStore.Reservations {
		if rsvt.Reservation.ID == id {
			break
		}
	}

	if r == nil {
		http.Error(w, fmt.Sprintf("reservation %s not found", id), http.StatusNotFound)
		return
	}

	rsvt.Deleted = true

	w.WriteHeader(http.StatusOK)

}

func deleteReservation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	provStore.Lock()
	defer provStore.Unlock()

	w.Header().Add("content-type", "application/json")

	for _, r := range provStore.Reservations {
		if r.Reservation.ID == id {

			r.Reservation.ToDelete = true

			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}
