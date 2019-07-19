package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func registerNode(w http.ResponseWriter, r *http.Request) {
	log.Println("node register request received")

	defer r.Body.Close()

	info := NodeInfo{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	i, ok := nodeStore[info.NodeID]
	if !ok {
		nodeStore[info.NodeID] = &info
	} else {
		i.NodeID = info.NodeID
		i.FarmID = info.FarmID
	}

	w.WriteHeader(http.StatusCreated)
}

func nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	exitIface, ok := exitIfaces[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	output := struct {
		Master  string
		IPv4    string
		IPv6    string
		GW4     string
		GW6     string
		Version int
	}{}

	output.Master = exitIface.Master
	output.Version = exitIface.Version
	if exitIface.IPv4 != nil {
		output.IPv4 = exitIface.IPv4.String()
	}
	if exitIface.IPv6 != nil {
		output.IPv6 = exitIface.IPv6.String()
	}
	if exitIface.GW4 != nil {
		output.GW4 = exitIface.GW4.String()
	}
	if exitIface.GW6 != nil {
		output.GW6 = exitIface.GW6.String()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&output)
}

func listNodes(w http.ResponseWriter, r *http.Request) {
	var identities = make([]*NodeInfo, 0, len(nodeStore))
	for _, info := range nodeStore {
		identities = append(identities, info)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(identities)
}

func registerFarm(w http.ResponseWriter, r *http.Request) {
	log.Println("farm register request received")

	defer r.Body.Close()

	info := FarmInfo{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	farmStore[info.ID] = &info
	w.WriteHeader(http.StatusCreated)
}

func listFarm(w http.ResponseWriter, r *http.Request) {
	var farms = make([]*FarmInfo, 0, len(farmStore))
	for _, info := range farmStore {
		farms = append(farms, info)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farms)
}
