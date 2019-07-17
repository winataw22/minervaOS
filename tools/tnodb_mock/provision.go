package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules/provision"
)

type reservation struct {
	Reservation *provision.Reservation
	Sent        bool
}

var reservationStore map[string][]*reservation
var reservationMu sync.Mutex

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

	reservationMu.Lock()
	defer reservationMu.Unlock()
	reservationStore[nodeID] = append(reservationStore[nodeID], &reservation{Reservation: res})
	w.WriteHeader(http.StatusCreated)
}

func getReservations(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]

	_, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node %s not found", nodeID), http.StatusNotFound)
		return
	}

	// start long polling
	timeout := time.Now().Add(time.Second * 10)
	output := []*provision.Reservation{}
	for {
		output = getRes(nodeID)
		if len(output) > 0 {
			break
		}

		if time.Now().After(timeout) {
			break
		}
		time.Sleep(time.Second)
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(output); err != nil {
		log.Printf("error encoding empty reservation slice: %v", err)
		return
	}
}

func getRes(nodeID string) []*provision.Reservation {
	output := []*provision.Reservation{}

	reservationMu.Lock()
	defer reservationMu.Unlock()

	rs, ok := reservationStore[nodeID]
	if ok {
		for _, res := range rs {
			if !res.Sent {
				output = append(output, res.Reservation)
			}
			res.Sent = true
		}
	}
	return output
}
