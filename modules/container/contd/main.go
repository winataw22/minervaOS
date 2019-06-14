package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/container"
)

const module = "container"

func main() {
	var (
		moduleRoot    string
		msgBrokerCon  string
		containerdCon string
		workerNr      uint
	)

	flag.StringVar(&moduleRoot, "root", "/var/modules/containerd", "root working directory of the module")
	flag.StringVar(&msgBrokerCon, "broker", "tcp://localhost:6379", "connection string to the message broker")
	flag.StringVar(&containerdCon, "containerd", "/run/containerd/containerd.sock", "connection string to containerd")
	flag.UintVar(&workerNr, "workers", 1, "number of workers")

	flag.Parse()

	if err := os.MkdirAll(moduleRoot, 0750); err != nil {
		log.Fatal().Msgf("fail to create module root: %s", err)
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	containerd := container.New(moduleRoot, containerdCon)

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, containerd)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting containerd module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
