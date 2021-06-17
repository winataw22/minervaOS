package flistd

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/flist"
)

const (
	module = "flist"

	cacheAge     = time.Hour * 24 * 3 // 3 days
	cacheCleanup = time.Hour * 24
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "flistd",
	Usage: "handles mounting of flists",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/flistd",
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

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	storage := stubs.NewStorageModuleStub(redis)

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	mod := flist.New(moduleRoot, storage)
	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, mod)

	ctx, _ := utils.WithSignal(context.Background())

	if cleaner, ok := mod.(flist.Cleaner); ok {
		//go cleaner.MountsCleaner(ctx, time.Minute)
		go cleaner.CacheCleaner(ctx, cacheCleanup, cacheAge)
	}

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting flist module")

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
