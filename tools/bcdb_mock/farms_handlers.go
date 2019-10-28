package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"

	"github.com/gorilla/mux"
)

func (s *farmStore) registerFarm(w http.ResponseWriter, r *http.Request) {
	log.Println("farm register request received")

	defer r.Body.Close()

	info := directory.TfgridFarm1{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Add(info); err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *farmStore) listFarm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s.List())
}

func (s *farmStore) cockpitListFarm(w http.ResponseWriter, r *http.Request) {
	x := struct {
		Farms []*directory.TfgridFarm1 `json:"farms"`
	}{s.List()}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(x)
}

func (s *farmStore) getFarm(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["farm_id"]
	farm, err := s.Get(name)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farm)
}
