package contd

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/container"
	"github.com/threefoldtech/zos/pkg/utils"
)

const module = "container"

// Module is contd entry point
var Module cli.Command = cli.Command{
	Name:  "contd",
	Usage: "handles containers creations",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/contd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.StringFlag{
			Name:  "congainerd",
			Usage: "connection string to containerd `CONTAINERD`",
			Value: "/run/containerd/containerd.sock",
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
		moduleRoot    string = cli.String("root")
		msgBrokerCon  string = cli.String("broker")
		workerNr      uint   = cli.Uint("workers")
		containerdCon string = cli.String("containerd")
	)

	// wait for shim-logs to be available before starting
	log.Info().Msg("wait for shim-logs binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 //forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("shim-logs")
		return err
		// return fmt.Errorf("wait forever")
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("shim-logs binary not found, retying in %s", d.String())
	})

	if err := os.MkdirAll(moduleRoot, 0750); err != nil {
		return errors.Wrap(err, "fail to create module root")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	containerd := container.New(client, moduleRoot, containerdCon)

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, containerd)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting containerd module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// start watching for events
	go containerd.Watch(ctx)

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
