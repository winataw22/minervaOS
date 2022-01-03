package qsfsd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/qsfsd"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const (
	module = "qsfsd"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "qsfsd",
	Usage: "manage qsfsd",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/qsfsd",
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

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	ctx, cancel := utils.WithSignal(cli.Context)
	defer cancel()

	mod, err := qsfsd.New(ctx, client, moduleRoot)
	if err != nil {
		return errors.Wrap(err, "failed to construct qsfsd object")
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, mod)
	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting qsfsd module")

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
