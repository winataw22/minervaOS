package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func registerNode(w http.ResponseWriter, r *http.Request) {
	log.Println("node register request received")

	defer r.Body.Close()

	n := node{}
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("node registered: %+v\n", n)

	i, ok := nodeStore[n.NodeID]
	if !ok || i.Node == nil {
		nodeStore[n.NodeID] = &n

	} else {
		i.NodeID = n.NodeID
		i.FarmID = n.FarmID
		i.Version = n.Version
	}

	w.WriteHeader(http.StatusCreated)
}

func nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	node, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&node); err != nil {
		log.Printf("error writing node: %v", err)
	}
}

func listNodes(w http.ResponseWriter, r *http.Request) {
	var nodes = make([]*types.Node, 0, len(nodeStore))
	farm := r.URL.Query().Get("farm")

	for _, node := range nodeStore {
		if node == nil {
			continue
		}

		if farm != "" && node.FarmID != farm {
			continue
		}
		nodes = append(nodes, node.Node)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(nodes)
}

func registerFarm(w http.ResponseWriter, r *http.Request) {
	log.Println("farm register request received")

	defer r.Body.Close()

	info := farmInfo{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	farmStore[info.ID] = &info
	w.WriteHeader(http.StatusCreated)
}

func listFarm(w http.ResponseWriter, r *http.Request) {
	var farms = make([]*farmInfo, 0, len(farmStore))
	for _, info := range farmStore {
		farms = append(farms, info)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farms)
}

func getFarm(w http.ResponseWriter, r *http.Request) {
	farmID := mux.Vars(r)["farm_id"]
	farm, ok := farmStore[farmID]
	if !ok {
		http.Error(w, fmt.Sprintf("farm %s not found", farmID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farm)
}

func registerCapacity(w http.ResponseWriter, r *http.Request) {
	x := struct {
		Capacity capacity.Capacity `json:"capacity,omitempty"`
		DMI      *dmi.DMI          `json:"dmi,omitempty"`
	}{}

	nodeID := mux.Vars(r)["node_id"]
	fmt.Println("search node", nodeID)
	node, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Capacity = x.Capacity

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
