package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules/network/types"
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
		node.PublicConfig = &types.PubIface{}
	}

	node.PublicConfig.Type = types.MacVlanIface //TODO: change me once we support other types
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
