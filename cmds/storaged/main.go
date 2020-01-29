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

	storageModule, err := storage.New()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing storage module")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	vdiskModule, err := storage.NewVDiskModule(storageModule)
	if err != nil {
		log.Fatal().Err(err)
	}

	server.Register(zbus.ObjectID{Name: "storage", Version: "0.0.1"}, storageModule)
	server.Register(zbus.ObjectID{Name: "vdisk", Version: "0.0.1"}, vdiskModule)

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
