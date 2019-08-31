package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules/network"
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

	ifaces := []*network.IfaceInfo{}
	if err := json.NewDecoder(r.Body).Decode(&ifaces); err != nil {
		log.Printf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("network interfaces received", ifaces)

	nodeStore[nodeID].Ifaces = ifaces
	w.WriteHeader(http.StatusCreated)
}

func chooseExit(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	node, ok := nodeStore[nodeID]
	if !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	farm, ok := farmStore[node.FarmID]
	if !ok {
		http.Error(w, fmt.Sprintf("farm id %s not found", node.FarmID), http.StatusNotFound)
		return
	}

	// mark the node as possible exit node
	node.ExitNode = true

	// add the node id to the list of possible exit node of the farm
	var found = false
	for _, nodeID := range farm.ExitNodes {
		if nodeID == node.NodeID {
			found = true
			break
		}
	}
	if !found {
		farm.ExitNodes = append(farm.ExitNodes, node.NodeID)
	}

	w.WriteHeader(http.StatusCreated)
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

	input := struct {
		Iface string   `json:"iface"`
		IPs   []string `json:"ips"`
		GWs   []string `json:"gateways"`
		// Type todo allow to chose type of connection
	}{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if node.PublicConfig == nil {
		node.PublicConfig = &network.PubIface{}
	}

	node.PublicConfig.Type = network.MacVlanIface //TODO: change me once we support other types
	node.PublicConfig.Master = input.Iface
	for i := range input.IPs {
		ip, ipnet, err := net.ParseCIDR(input.IPs[i])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ipnet.IP = ip

		if ip.To4() != nil {
			node.PublicConfig.IPv4 = ipnet
		} else if ip.To16() != nil {
			node.PublicConfig.IPv6 = ipnet
		}

		gw := net.ParseIP(input.GWs[i])
		if gw.To4() != nil {
			node.PublicConfig.GW4 = gw
		} else if gw.To16() != nil {
			node.PublicConfig.GW6 = gw
		}
	}
	node.PublicConfig.Version++

	w.WriteHeader(http.StatusCreated)
}

func registerAlloc(w http.ResponseWriter, r *http.Request) {
	log.Println("prefix register request received")

	defer r.Body.Close()

	type tmp struct {
		FarmerID string `json:"farmer_id"`
		Prefix   string `json:"allocation"`
	}
	req := tmp{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	allocStore.Lock()
	defer allocStore.Unlock()
	if _, ok := farmStore[req.FarmerID]; !ok {
		http.Error(w,
			fmt.Sprintf("farmer %s does not exist, register this farmer fist", req.FarmerID),
			http.StatusBadRequest)
		return
	}

	_, prefix, err := net.ParseCIDR(req.Prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	allocStore.Allocations[req.FarmerID] = &allocation{
		Allocation: prefix,
	}
	w.WriteHeader(http.StatusCreated)
}

func listAlloc(w http.ResponseWriter, r *http.Request) {
	allocStore.Lock()
	defer allocStore.Unlock()
	allocs := make([]string, 0, len(allocStore.Allocations))
	for _, prefix := range allocStore.Allocations {
		allocs = append(allocs, prefix.Allocation.String())
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(allocs)
}

func getAlloc(w http.ResponseWriter, r *http.Request) {
	farmID, ok := mux.Vars(r)["farm_id"]
	if !ok {
		http.Error(w, "missing farm_id", http.StatusBadRequest)
		return
	}

	if _, ok := farmStore[farmID]; !ok {
		http.Error(w, "farm not found", http.StatusNotFound)
		return
	}

	alloc, farmAlloc, err := requestAllocation(farmID, allocStore)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Alloc     string `json:"allocation"`
		FarmAlloc string `json:"farm_allocation"`
	}{
		Alloc:     alloc.String(),
		FarmAlloc: farmAlloc.String(),
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&data)
}
