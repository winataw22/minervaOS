package main

import (
	"context"
	"flag"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/flist"
	"github.com/threefoldtech/zos/pkg/version"
)

const module = "flist"

func main() {
	app.Initialize()

	var (
		moduleRoot   string
		msgBrokerCon string
		workerNr     uint
		ver          bool
	)

	flag.StringVar(&moduleRoot, "root", "/var/cache/modules/flistd", "root working directory of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "number of workers")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}
	storage := stubs.NewStorageModuleStub(redis)

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	flist := flist.New(moduleRoot, storage)
	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, flist)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting flist module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}
