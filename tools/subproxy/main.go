package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/substrate"
)

func main() {
	var (
		url     string
		address string
	)

	flag.StringVar(&url, "substrate", "", "websocket url to substrate")
	flag.StringVar(&address, "address", ":9950", "listening address")

	flag.Parse()

	client, err := substrate.NewSubstrate(url)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to substrate")
	}

	router := mux.NewRouter()

	router.Path("/twin/{id}").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid twin id '%s': %s", idStr, err)
			return
		}
		twin, err := client.GetTwin(uint32(id))
		if errors.Is(err, substrate.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to query twin: %s", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(twin); err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
	})

	server := http.Server{
		Addr:    address,
		Handler: handlers.LoggingHandler(os.Stdout, router),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Error().Err(err).Msg("server exited")
	}
}
