package gateway

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gateway"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const (
	module = "gateway"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "gateway",
	Usage: "manage web gateway proxy",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/gateway",
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

	mod, err := gateway.New(cli.Context, client, moduleRoot)
	if err != nil {
		return errors.Wrap(err, "failed to construct gateway object")
	}
	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, mod)

	ctx, cancel := utils.WithSignal(context.Background())
	defer cancel()

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting gateway module")

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
