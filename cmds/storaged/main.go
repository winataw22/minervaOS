package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/storage"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "storage"
)

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		workerNr     uint
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", redisSocket, "Connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "Number of workers")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	storage, err := storage.New()
	if err != nil {
		log.Fatal().Msgf("Error initializing storage module: %s", err)
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, storage)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting storaged module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}
