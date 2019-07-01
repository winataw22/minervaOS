package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"
)

type NodeInfo struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`

	Ifaces []ifaceInfo
}
type ifaceInfo struct {
	Name    string   `json:"name"`
	Addrs   []string `json:"addrs"`
	Gateway []string `json:"gateway"`
}

type FarmInfo struct {
	ID   string `json:"farm_id"`
	Name string `json:"name"`
}

var (
	nodeStore    map[string]*NodeInfo
	exitNodes    map[string]*network.ExitIface
	farmStore    map[string]*FarmInfo
	networkStore map[string]*modules.Network
)

type allocationStore struct {
	sync.Mutex
	Allocations map[string]*Allocation
}

type Allocation struct {
	Allocation *net.IPNet
	SubNetUsed []uint64
}

var allocStore *allocationStore

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
	if _, ok := nodeStore[nodeID]; !ok {
		http.Error(w, fmt.Sprintf("node id %s not found", nodeID), http.StatusNotFound)
		return
	}

	type input struct {
		Iface string `json:"iface"`
		IP    string `json:"ip"`
		GW    string `json:"gateway"`
		// Type todo allow to chose type of connection
	}
	i := input{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO verifiy the iface sent by user actually exists
	var exitIface *network.ExitIface
	exitIface, ok := exitNodes[nodeID]
	if !ok {
		exitIface = &network.ExitIface{}
		exitNodes[nodeID] = exitIface
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

func nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	exitIface, ok := exitNodes[nodeID]
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

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&output)
}

func listNodes(w http.ResponseWriter, r *http.Request) {
	var identities = make([]*NodeInfo, 0, len(nodeStore))
	for _, info := range nodeStore {
		identities = append(identities, info)
	}

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

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farms)
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
	json.NewEncoder(w).Encode(network)
}

func createNetwork(w http.ResponseWriter, r *http.Request) {
	network := modules.Network{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("%+v", network)
	networkStore[string(network.NetID)] = &network
	fmt.Println(networkStore)

	w.WriteHeader(http.StatusCreated)
}

func publishWGKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	netid := vars["netid"]
	nodeid := vars["nodeid"]

	network, ok := networkStore[netid]
	if !ok {
		http.Error(w, fmt.Sprintf("network not found"), http.StatusNotFound)
		return
	}

	var netR *modules.NetResource
	for _, res := range network.Resources {
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
	for _, res := range network.Resources {
		for _, peer := range res.Peers {
			if peer.Prefix.String() == netR.Prefix.String() {
				peer.Connection.Key = key.Key
			}
		}
	}

	networkStore[string(network.NetID)] = network

	w.WriteHeader(http.StatusCreated)
}

var listen string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	nodeStore = make(map[string]*NodeInfo)
	exitNodes = make(map[string]*network.ExitIface)
	farmStore = make(map[string]*FarmInfo)
	networkStore = make(map[string]*modules.Network)
	allocStore = &allocationStore{Allocations: make(map[string]*Allocation)}

	router := mux.NewRouter()

	router.HandleFunc("/nodes", registerNode).Methods("POST")
	router.HandleFunc("/nodes/{node_id}", nodeDetail).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/interfaces", registerIfaces).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/enable_exit", chooseExit).Methods("POST")
	router.HandleFunc("/nodes", listNodes).Methods("GET")
	router.HandleFunc("/farms", registerFarm).Methods("POST")
	router.HandleFunc("/farms", listFarm).Methods("GET")
	router.HandleFunc("/allocations", registerAlloc).Methods("POST")
	router.HandleFunc("/allocations", listAlloc).Methods("GET")
	router.HandleFunc("/allocations/{farm_id}", getAlloc).Methods("GET")
	router.HandleFunc("/networks/{netid}", getNetwork).Methods("GET")
	router.HandleFunc("/networks", createNetwork).Methods("POST")
	router.HandleFunc("/networks/{netid}/{nodeid}/wgkeys", publishWGKey).Methods("POST")

	log.Printf("start on %s\n", listen)
	http.ListenAndServe(listen, router)
}
