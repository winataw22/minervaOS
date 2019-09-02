package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules/provision"
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
	w.WriteHeader(http.StatusCreated)
}

func pollReservations(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]

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
		output = getRes(nodeID, all)
	} else {
		// otherwise start long polling
		timeout := time.Now().Add(time.Second * 20)
		for {
			output = getRes(nodeID, all)
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

func getRes(nodeID string, all bool) []*provision.Reservation {
	output := []*provision.Reservation{}

	provStore.Lock()
	defer provStore.Unlock()

	for _, r := range provStore.Reservations {
		// skip reservation aimed at another node
		if r.NodeID != nodeID {
			continue
		}
		// if we are long polling, only return the new reservation
		if !all && r.Reservation.Result != nil || r.Reservation.Expired() {
			continue
		}
		output = append(output, r.Reservation)
	}

	return output
}

func getReservation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	provStore.Lock()
	defer provStore.Unlock()

	w.Header().Add("content-type", "application/json")

	for _, r := range provStore.Reservations {
		if r.Reservation.ID == id {
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(r.Reservation); err != nil {
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
	rsvt.Reservation.Result = result

	w.WriteHeader(http.StatusOK)
}
