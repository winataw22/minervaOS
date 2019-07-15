package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"
)

func registerIfaces(w http.ResponseWriter, r *http.Request) {
	log.Println("ifaces register request received")

	nodeID := mux.Vars(r)["node_id"]
	if _, ok := nodeStore[nodeID]; !ok {
		err := fmt.Errorf("node id %s not found", nodeID)
		log.Printf("node not found %v", nodeID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	defer r.Body.Close()

	ifaces := []ifaceInfo{}
	if err := json.NewDecoder(r.Body).Decode(&ifaces); err != nil {
		log.Printf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("ifaces received", ifaces)

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

	i := struct {
		Iface string `json:"iface,omitempty"`
		IP    string `json:"ip,omitempty"`
		GW    string `json:"gateway,omitempty"`
		// Type todo allow to chose type of connection
	}{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO verify the iface sent by user actually exists
	var exitIface *network.PubIface
	exitIface, ok = exitIfaces[nodeID]
	if !ok {
		exitIface = &network.PubIface{}
		exitIfaces[nodeID] = exitIface
	}

	exitIface.Version++
	exitIface.Master = i.Iface
	ip, ipnet, err := net.ParseCIDR(i.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ipnet.IP = ip

	if ip.To4() != nil {
		exitIface.IPv4 = ipnet
	} else if ip.To16() != nil {
		exitIface.IPv6 = ipnet
	}

	gw := net.ParseIP(i.GW)
	if gw.To4() != nil {
		exitIface.GW4 = gw
	} else if gw.To16() != nil {
		exitIface.GW6 = gw
	}

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

	allocStore.Allocations[req.FarmerID] = &Allocation{
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

	alloc, err := requestAllocation(farmID, allocStore)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Alloc string `json:"allocation"`
	}{alloc.String()}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&data)
}

func getNetwork(w http.ResponseWriter, r *http.Request) {
	netid := mux.Vars(r)["netid"]

	network, ok := networkStore[netid]
	if !ok {
		http.Error(w, fmt.Sprintf("network not found"), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(network.Network)
}

func createNetwork(w http.ResponseWriter, r *http.Request) {

	networkReq := struct {
		ExitFarm string `json:"exit_farm"`
	}{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&networkReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	farm, ok := farmStore[networkReq.ExitFarm]
	if !ok {
		http.Error(w, fmt.Sprintf("farm %s not found", networkReq.ExitFarm), http.StatusNotFound)
		return
	}

	if len(farm.ExitNodes) <= 0 {
		http.Error(w, fmt.Sprintf("farm %s doesn't have any exit node configured", networkReq.ExitFarm), http.StatusNotFound)
		return
	}

	allocZero, allocSize, err := getNetworkZero(networkReq.ExitFarm, allocStore)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	alloc, err := requestAllocation(networkReq.ExitFarm, allocStore)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	exitNodeID := farm.ExitNodes[0]
	exitNibble := meaningfullNibble(alloc, allocSize)

	netid, err := uuid.NewRandom()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ipZero := netZeroIP(allocZero, alloc, allocSize)

	exitPeer := &modules.Peer{
		Type:   modules.ConnTypeWireguard,
		Prefix: alloc,
		Connection: modules.Wireguard{
			IP:   ipZero,
			Port: 1600,
			Key:  "",
		},
	}
	linkLocal := &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::%s", exitNibble.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
	exitResource := &modules.NetResource{
		NodeID: &modules.NodeID{
			ID:             exitNodeID,
			FarmerID:       networkReq.ExitFarm,
			ReachabilityV6: modules.ReachabilityV6Public,
			ReachabilityV4: modules.ReachabilityV4Public,
		},
		Prefix:    alloc,
		LinkLocal: linkLocal,
		Peers:     []*modules.Peer{exitPeer},
		ExitPoint: true,
	}

	exitPoint := &modules.ExitPoint{
		Ipv6Conf: &modules.Ipv6Conf{
			Addr:    linkLocal,
			Gateway: net.ParseIP("fe80::1"),
			Iface:   "public",
		},
	}
	network := &modules.Network{
		NetID: modules.NetID(netid.String()),
		Resources: []*modules.NetResource{
			exitResource,
		},
		PrefixZero: allocZero,
		Exit:       exitPoint,
		Version:    0,
	}

	networkStore[string(network.NetID)] = &NetworkInfo{
		Network:   network,
		ExitPoint: exitPoint,
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(network); err != nil {
		log.Println("error while marshalling network into json")
	}
}

func publishWGKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	netid := vars["netid"]
	nodeid := vars["nodeid"]

	ni, ok := networkStore[netid]
	if !ok {
		http.Error(w, fmt.Sprintf("network not found"), http.StatusNotFound)
		return
	}

	var netR *modules.NetResource
	for _, res := range ni.Network.Resources {
		if res.NodeID.ID == nodeid {
			netR = res
			break
		}
	}
	if netR == nil {
		http.Error(w, fmt.Sprintf("node ID %s not found in network %s", nodeid, netid), http.StatusNotFound)
		return
	}

	key := struct {
		Key string `json:"key"`
	}{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&key); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: ensure the key is coming from the node and not an attacker

	// update all the peers arrays in all the network resources
	// with the key published by the node
	for _, res := range ni.Network.Resources {
		for _, peer := range res.Peers {
			if peer.Prefix.String() == netR.Prefix.String() {
				peer.Connection.Key = key.Key
			}
		}
	}

	ni.Network.Version++
	networkStore[string(ni.Network.NetID)] = ni

	w.WriteHeader(http.StatusCreated)
}

type nibble []byte

func (n nibble) Hex() string {
	if len(n) == 2 {
		return fmt.Sprintf("%x", n)
	}
	if len(n) == 4 {
		return fmt.Sprintf("%x:%x", n[:2], n[2:])
	}
	panic("wrong nibble size")
}

func meaningfullNibble(prefix *net.IPNet, size int) nibble {
	var n []byte

	if size < 48 {
		n = []byte(prefix.IP)[4:8]
	} else {
		n = []byte(prefix.IP)[6:8]
	}
	return nibble(n)
}

func netZeroIP(netZero *net.IPNet, alloc *net.IPNet, allocSize int) net.IP {
	nibble := meaningfullNibble(alloc, allocSize)

	ipZero := make([]byte, net.IPv6len)
	copy(ipZero[:], netZero.IP)
	copy(ipZero[net.IPv6len-len(nibble):], nibble)
	return net.IP(ipZero[:])
}
