package storaged

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/storage"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "storage"
)

// Module is module entry point
var Module cli.Command = cli.Command{
	Name:  "storaged",
	Usage: "handles and manages disks and volumes creation",
	Flags: []cli.Flag{
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
		&cli.UintFlag{
			Name:  "expvar-port",
			Usage: "port to host expvar variables on `N`",
			Value: 28682,
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		workerNr     uint   = cli.Uint("workers")
		expvarPort   uint   = cli.Uint("expvar-port")
	)

	storageModule, err := storage.New()
	if err != nil {
		return errors.Wrap(err, "failed to initialize storage module")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
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
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
