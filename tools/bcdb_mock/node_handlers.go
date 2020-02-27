package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/threefoldtech/zos/pkg/capacity"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

func (s *nodeStore) registerNode(w http.ResponseWriter, r *http.Request) {
	log.Println("node register request received")

	defer r.Body.Close()

	n := directory.TfgridNode2{}
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	if err := s.Add(n); err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	log.Printf("node registered: %+v\n", n)

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	node, err := s.Get(nodeID)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&node); err != nil {
		log.Printf("error writing node: %v", err)
	}
}

func (s *nodeStore) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.List()
	sFarm := r.URL.Query().Get("farm")

	var (
		farm uint64
		err  error
	)

	if sFarm != "" {
		farm, err = strconv.ParseUint(sFarm, 10, 64)
		if err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}
	}

	for i, node := range nodes {
		if node == nil {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}

		if farm != 0 && uint64(node.FarmID) != farm {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(nodes)
}

func (s *nodeStore) cockpitListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.List()
	sFarm := r.URL.Query().Get("farm")

	var (
		farm uint64
		err  error
	)

	if sFarm != "" {
		farm, err = strconv.ParseUint(sFarm, 10, 64)
		if err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}
	}

	for i, node := range nodes {
		if node == nil {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}

		if farm != 0 && uint64(node.FarmID) != farm {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}
	}

	x := struct {
		Node []*directory.TfgridNode2 `json:"nodes"`
	}{nodes}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(x)
}

func (s *nodeStore) registerCapacity(w http.ResponseWriter, r *http.Request) {
	x := struct {
		Capacity   directory.TfgridNodeResourceAmount1 `json:"capacity,omitempty"`
		DMI        dmi.DMI                             `json:"dmi,omitempty"`
		Disks      capacity.Disks                      `json:"disks,omitempty"`
		Hypervisor []string                            `json:"hypervisor,omitempty"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.updateTotalCapacity(nodeID, x.Capacity); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	if err := s.StoreProof(nodeID, x.DMI, x.Disks, x.Hypervisor); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *nodeStore) registerIfaces(w http.ResponseWriter, r *http.Request) {
	log.Println("network interfaces register request received")

	defer r.Body.Close()

	input := []*types.IfaceInfo{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Print(err.Error())
		httpError(w, err, http.StatusBadRequest)
		return
	}

	log.Println("network interfaces received", input)
	ifaces := make([]directory.TfgridNodeIface1, len(input))
	for i, iface := range input {
		ifaces[i].Gateway = iface.Gateway
		ifaces[i].Name = iface.Name
		for _, r := range iface.Addrs {
			ifaces[i].Addrs = append(ifaces[i].Addrs, r.ToSchema())
		}
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetInterfaces(nodeID, ifaces); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) configurePublic(w http.ResponseWriter, r *http.Request) {
	iface := types.PubIface{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	cfg := directory.TfgridNodePublicIface1{
		Gw4:    iface.GW4,
		Gw6:    iface.GW6,
		Ipv4:   iface.IPv4.ToSchema(),
		Ipv6:   iface.IPv6.ToSchema(),
		Master: iface.Master,
		Type:   directory.TfgridNodePublicIface1TypeMacvlan,
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetPublicConfig(nodeID, cfg); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) registerPorts(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	input := struct {
		Ports []uint `json:"ports"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	fmt.Println("wireguard ports received", input.Ports)

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetWGPorts(nodeID, input.Ports); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *nodeStore) updateUptimeHandler(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	input := struct {
		Uptime uint64
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	nodeID := mux.Vars(r)["node_id"]
	fmt.Printf("node uptime received %s %d\n", nodeID, input.Uptime)

	if err := s.updateUptime(nodeID, int64(input.Uptime)); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *nodeStore) updateUsedResources(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	u := struct {
		SRU int64 `json:"sru,omitempty"`
		HRU int64 `json:"hru,omitempty"`
		MRU int64 `json:"mru,omitempty"`
		CRU int64 `json:"cru,omitempty"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	nodeID := mux.Vars(r)["node_id"]

	usedRescources := directory.TfgridNodeResourceAmount1{
		Cru: int64(u.CRU),
		Sru: int64(u.SRU),
		Hru: int64(u.HRU),
		Mru: int64(u.MRU),
	}

	if err := s.updateReservedCapacity(nodeID, usedRescources); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
