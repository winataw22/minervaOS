package main

import (
	"context"
	"flag"
	"os"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/vm"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/version"
)

const module = "vmd"

func main() {
	app.Initialize()

	var (
		moduleRoot   string
		msgBrokerCon string
		workerNr     uint
		ver          bool
	)

	flag.StringVar(&moduleRoot, "root", "/var/cache/modules/vmd", "root working directory of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "number of workers")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		log.Fatal().Err(err).Str("root", moduleRoot).Msg("Failed to create module root")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	mod, err := vm.NewVMModule(moduleRoot)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create a new instance of manager")
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, mod)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting vmd module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}
