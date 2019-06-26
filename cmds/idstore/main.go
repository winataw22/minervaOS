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
)

type NodeInfo struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`
}

type FarmInfo struct {
	ID   string `json:"farm_id"`
	Name string `json:"name"`
}

var nodeStore map[string]*NodeInfo
var farmStore map[string]*FarmInfo

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

	nodeStore[info.NodeID] = &info
	w.WriteHeader(http.StatusCreated)
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

var listen string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	nodeStore = make(map[string]*NodeInfo)
	farmStore = make(map[string]*FarmInfo)
	allocStore = &allocationStore{Allocations: make(map[string]*Allocation)}

	router := mux.NewRouter()

	router.HandleFunc("/nodes", registerNode).Methods("POST")
	router.HandleFunc("/nodes", listNodes).Methods("GET")
	router.HandleFunc("/farms", registerFarm).Methods("POST")
	router.HandleFunc("/farms", listFarm).Methods("GET")
	router.HandleFunc("/allocations", registerAlloc).Methods("POST")
	router.HandleFunc("/allocations", listAlloc).Methods("GET")
	router.HandleFunc("/allocations/{farm_id}", getAlloc).Methods("GET")

	log.Printf("start on %s\n", listen)
	http.ListenAndServe(listen, router)
}
