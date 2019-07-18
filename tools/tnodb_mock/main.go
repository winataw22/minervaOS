package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/handlers"
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

func (i ifaceInfo) DefaultIP() (net.IP, error) {
	if len(i.Gateway) <= 0 {
		return nil, fmt.Errorf("iterface has not gateway")
	}

	for _, addr := range i.Addrs {
		ip, _, err := net.ParseCIDR(addr)
		if err != nil {
			continue
		}
		if ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.To4() != nil {
			continue
		}

		if ip.To16() != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no ipv6 address with default gateway")
}

type FarmInfo struct {
	ID        string   `json:"farm_id"`
	Name      string   `json:"name"`
	ExitNodes []string `json:"exit_nodes"`
}

type NetworkInfo struct {
	Network   *modules.Network
	ExitPoint *modules.ExitPoint
}

var (
	nodeStore    map[string]*NodeInfo
	exitIfaces   map[string]*network.PubIface
	farmStore    map[string]*FarmInfo
	networkStore map[string]*NetworkInfo
	allocStore   *allocationStore
)

type allocationStore struct {
	sync.Mutex
	Allocations map[string]*Allocation `json:"allocations"`
}

type Allocation struct {
	Allocation *net.IPNet
	SubNetUsed []uint64
}

var listen string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	nodeStore = make(map[string]*NodeInfo)
	exitIfaces = make(map[string]*network.PubIface)
	farmStore = make(map[string]*FarmInfo)
	networkStore = make(map[string]*NetworkInfo)
	allocStore = &allocationStore{Allocations: make(map[string]*Allocation)}
	reservationStore = make(map[string][]*reservation)

	if err := load(); err != nil {
		log.Fatalf("failed to load data: %v\n", err)
	}
	defer func() {
		if err := save(); err != nil {
			log.Printf("failed to save data: %v\n", err)
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/nodes", registerNode).Methods("POST")
	router.HandleFunc("/nodes/{node_id}", nodeDetail).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/interfaces", registerIfaces).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/configure_public", configurePublic).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/select_exit", chooseExit).Methods("POST")
	router.HandleFunc("/nodes", listNodes).Methods("GET")
	router.HandleFunc("/farms", registerFarm).Methods("POST")
	router.HandleFunc("/farms", listFarm).Methods("GET")

	router.HandleFunc("/allocations", registerAlloc).Methods("POST")
	router.HandleFunc("/allocations", listAlloc).Methods("GET")
	router.HandleFunc("/allocations/{farm_id}", getAlloc).Methods("GET")
	router.HandleFunc("/networks", createNetwork).Methods("POST")
	router.HandleFunc("/networks/{netid}", getNetwork).Methods("GET")
	router.HandleFunc("/networks/{netid}", addNode).Methods("POST")
	router.HandleFunc("/networks/{netid}/user", addUser).Methods("POST")
	router.HandleFunc("/networks/{node_id}/versions", getNetworksVersion).Methods("GET")
	router.HandleFunc("/networks/{netid}/{nodeid}/wgkeys", publishWGKey).Methods("POST")

	router.HandleFunc("/reserve/{node_id}", reserve).Methods("POST")
	router.HandleFunc("/reserve/{node_id}", getReservations).Methods("GET")

	log.Printf("start on %s\n", listen)
	loggedRouter := handlers.LoggingHandler(os.Stderr, router)
	s := &http.Server{
		Addr:    listen,
		Handler: loggedRouter,
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go s.ListenAndServe()

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v\n", err)
	}
}

func save() error {
	stores := map[string]interface{}{
		"nodes":        nodeStore,
		"exits":        exitIfaces,
		"farms":        farmStore,
		"network":      networkStore,
		"allocations":  allocStore,
		"reservations": reservationStore,
	}
	for name, store := range stores {
		f, err := os.OpenFile(name+".json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(store); err != nil {
			return err
		}
	}
	return nil
}

func load() error {
	stores := map[string]interface{}{
		"nodes":        &nodeStore,
		"exits":        &exitIfaces,
		"farms":        &farmStore,
		"network":      &networkStore,
		"allocations":  &allocStore,
		"reservations": &reservationStore,
	}
	for name, store := range stores {
		f, err := os.OpenFile(name+".json", os.O_RDONLY, 0660)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(store); err != nil {
			return err
		}
	}
	return nil
}
