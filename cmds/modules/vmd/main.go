package vmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/vm"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const module = "vmd"

// Module entry point
var Module cli.Command = cli.Command{
	Name:  module,
	Usage: "handles virtual machines creation",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/vmd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.UintFlag{
			Name:  "workers",
			Usage: "number of workers `N`",
			Value: 1,
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		moduleRoot   string = cli.String("root")
		msgBrokerCon string = cli.String("broker")
		workerNr     uint   = cli.Uint("workers")
	)

	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		log.Fatal().Err(err).Str("root", moduleRoot).Msg("Failed to create module root")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	mod, err := vm.NewVMModule(client, moduleRoot)
	if err != nil {
		return errors.Wrap(err, "failed to create a new instance of manager")
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, mod)

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	mod.Monitor(ctx)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting vmd module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
