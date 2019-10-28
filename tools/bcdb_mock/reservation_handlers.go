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

func (s *reservationsStore) reserve(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]

	defer r.Body.Close()
	res := &provision.Reservation{}
	if err := json.NewDecoder(r.Body).Decode(res); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := provision.Verify(res); err != nil {
		errmsg := fmt.Sprintf("reservation signature invalid: %s", err.Error())
		http.Error(w, errmsg, http.StatusBadRequest)
		return
	}

	s.Add(nodeID, res)

	w.Header().Set("Location", "/reservations/"+res.ID)
	w.WriteHeader(http.StatusCreated)
}

func (s *reservationsStore) poll(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	var since time.Time
	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		// if since is not specificed, send all reservation since last hour
		since = time.Now().Add(-time.Hour)
	} else {
		timestamp, err := strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			http.Error(w, "since query argument format not valid", http.StatusBadRequest)
			return
		}
		since = time.Unix(timestamp, 0)
	}

	all, err := strconv.ParseBool(r.URL.Query().Get("all"))
	if err != nil {
		all = false
	}

	output := []*provision.Reservation{}
	if all {
		// just get all reservation for this nodeID
		output = s.GetReservations(nodeID, all, since)
	} else {
		// otherwise start long polling
		timeout := time.Now().Add(time.Second * 20)
		for {
			output = s.GetReservations(nodeID, all, since)
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

func (s *reservationsStore) get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	reservation, err := s.Get(id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(reservation.Reservation); err != nil {
		log.Printf("error during json encoding of reservation: %v", err)
	}
}

func (s *reservationsStore) putResult(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	reservation, err := s.Get(id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Add("content-type", "application/json")

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reservation.Result); err != nil {
		log.Printf("failed to decode reservation result: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *reservationsStore) putDeleted(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	reservation, err := s.Get(id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	reservation.Deleted = true

	w.WriteHeader(http.StatusOK)

}

func (s *reservationsStore) delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	w.Header().Add("content-type", "application/json")

	reservation, err := s.Get(id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	reservation.Reservation.ToDelete = true

	w.WriteHeader(http.StatusOK)
}
