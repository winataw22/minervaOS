package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/container"
	"github.com/threefoldtech/zosv2/modules/version"
)

const module = "container"

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		moduleRoot    string
		msgBrokerCon  string
		containerdCon string
		workerNr      uint
		ver           bool
	)

	flag.StringVar(&moduleRoot, "root", "/var/modules/containerd", "root working directory of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&containerdCon, "containerd", "/run/containerd/containerd.sock", "connection string to containerd")
	flag.UintVar(&workerNr, "workers", 1, "number of workers")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

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
