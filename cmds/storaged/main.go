package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/storage"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "storage"
)

func main() {
	var (
		msgBrokerCon string
		workerNr     uint
	)

	flag.StringVar(&msgBrokerCon, "broker", redisSocket, "Connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "Number of workers")

	flag.Parse()

	storage := storage.New()

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, storage)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting storaged module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}

	log.Warn().Msgf("Exiting storaged")
}
