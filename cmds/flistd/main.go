package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/flist"
)

var (
	moduleRoot   = flag.String("root", "/var/modules/flist", "root working directory of the module")
	msgBrokerCon = flag.String("broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	workerNr     = flag.Uint("workers", 1, "number of workers")
)

const module = "flist"

func main() {
	flag.Parse()

	server, err := zbus.NewRedisServer(module, *msgBrokerCon, *workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	flist := flist.New(*moduleRoot)
	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, flist)

	log.Info().
		Str("broker", *msgBrokerCon).
		Uint("worker nr", *workerNr).
		Msg("starting flist module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
