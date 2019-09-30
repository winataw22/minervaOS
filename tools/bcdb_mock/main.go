package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zosv2/modules/capacity"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/provision"
)

type farmInfo struct {
	ID        string   `json:"farm_id"`
	Name      string   `json:"name"`
	ExitNodes []string `json:"exit_nodes"`
}

type reservation struct {
	Reservation *provision.Reservation `json:"reservation"`
	Result      *provision.Result      `json:"result"`
	Deleted     bool                   `json:"deleted"`
	NodeID      string                 `json:"node_id"`
}

type provisionStore struct {
	sync.Mutex
	Reservations []*reservation `json:"reservations"`
}

type allocationStore struct {
	sync.Mutex
	Allocations map[string]*allocation `json:"allocations"`
}

type allocation struct {
	Allocation *net.IPNet
	SubNetUsed []uint64
}

type node struct {
	*types.Node
	capacity.Capacity
}

var (
	nodeStore  map[string]*node
	farmStore  map[string]*farmInfo
	allocStore *allocationStore
	provStore  *provisionStore
)

var listen string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	nodeStore = make(map[string]*node)
	farmStore = make(map[string]*farmInfo)
	allocStore = &allocationStore{Allocations: make(map[string]*allocation)}
	provStore = &provisionStore{Reservations: make([]*reservation, 0, 20)}

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
	router.HandleFunc("/nodes/{node_id}/ports", registerPorts).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/configure_public", configurePublic).Methods("POST")
	// router.HandleFunc("/nodes/{node_id}/select_exit", chooseExit).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/capacity", registerCapacity).Methods("POST")
	router.HandleFunc("/nodes", listNodes).Methods("GET")

	router.HandleFunc("/farms", registerFarm).Methods("POST")
	router.HandleFunc("/farms", listFarm).Methods("GET")
	router.HandleFunc("/farms/{farm_id}", getFarm).Methods("GET")

	// router.HandleFunc("/allocations", registerAlloc).Methods("POST")
	// router.HandleFunc("/allocations", listAlloc).Methods("GET")
	// router.HandleFunc("/allocations/{node_id}", getAlloc).Methods("GET")

	router.HandleFunc("/reservations/{node_id}", reserve).Methods("POST")
	router.HandleFunc("/reservations/{node_id}/poll", pollReservations).Methods("GET")
	router.HandleFunc("/reservations/{id}", getReservation).Methods("GET")
	router.HandleFunc("/reservations/{id}", reservationResult).Methods("PUT")
	router.HandleFunc("/reservations/{id}/deleted", reservationDeleted).Methods("PUT")
	router.HandleFunc("/reservations/{id}", deleteReservation).Methods("DELETE")

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
		"farms":        farmStore,
		"allocations":  allocStore,
		"reservations": provStore,
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
		"farms":        &farmStore,
		"allocations":  &allocStore,
		"reservations": &provStore,
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
