package main

import (
	"context"
	"flag"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/flist"
	"github.com/threefoldtech/zosv2/modules/version"
)

const module = "flist"

func main() {
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

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
