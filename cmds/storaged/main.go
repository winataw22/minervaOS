package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

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
		expvarPort   uint
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", redisSocket, "Connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "Number of workers")
	flag.UintVar(&expvarPort, "expvarPort", 28682, "Port to host expvar variables on")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	storageModule, err := storage.New()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage module")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: "storage", Version: "0.0.1"}, storageModule)

	vdiskModule, err := storage.NewVDiskModule(storageModule)
	if err != nil {
		log.Error().Err(err).Bool("limited-cache", app.CheckFlag(app.LimitedCache)).Msg("failed to initialize virtual disk module")
	} else {
		server.Register(zbus.ObjectID{Name: "vdisk", Version: "0.0.1"}, vdiskModule)
	}

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting storaged module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", expvarPort), http.DefaultServeMux); err != nil {
			log.Error().Err(err).Msg("Error starting http server")
		}
	}()

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}
