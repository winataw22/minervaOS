package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func registerIfaces(w http.ResponseWriter, r *http.Request) {
	log.Println("network interfaces register request received")

	nodeID := mux.Vars(r)["node_id"]
	if _, ok := nodeStore[nodeID]; !ok {
		err := fmt.Errorf("node id %s not found", nodeID)
		log.Printf("node not found %v", nodeID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	defer r.Body.Close()

	ifaces := []*types.IfaceInfo{}
	if err := json.NewDecoder(r.Body).Decode(&ifaces); err != nil {
		log.Printf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("network interfaces received", ifaces)

	nodeStore[nodeID].Ifaces = ifaces
	w.WriteHeader(http.StatusCreated)
}

func registerPorts(w http.ResponseWriter, r *http.Request) {

	nodeID := mux.Vars(r)["node_id"]
	if _, ok := nodeStore[nodeID]; !ok {
		err := fmt.Errorf("node id %s not found", nodeID)
		log.Printf("node not found %v", nodeID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	defer r.Body.Close()

	input := struct {
		Ports []uint `json:"ports"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("wireguard ports received", input.Ports)

	nodeStore[nodeID].WGPorts = input.Ports
	w.WriteHeader(http.StatusOK)
}

func configurePublic(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	node, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	if _, ok = farmStore[node.FarmID]; !ok {
		http.Error(w, fmt.Sprintf("farm id %s not found", node.FarmID), http.StatusNotFound)
		return
	}

	version := 0
	if node.PublicConfig != nil {
		version = node.PublicConfig.Version
	}
	node.PublicConfig = &types.PubIface{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(node.PublicConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node.PublicConfig.Type = types.MacVlanIface //TODO: change me once we support other types
	node.PublicConfig.Version = version + 1

	w.WriteHeader(http.StatusCreated)
}
