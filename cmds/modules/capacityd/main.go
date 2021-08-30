package capacityd

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const module = "monitor"

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "capacityd",
	Usage: "reports the node total resources",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
	},
	Action: action,
}

func mon(ctx context.Context, server zbus.Server) {
	system, err := monitord.NewSystemMonitor(2 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize system monitor")
	}
	host, err := monitord.NewHostMonitor(2 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize host monitor")
	}

	server.Register(zbus.ObjectID{Name: "host", Version: "0.0.1"}, host)
	server.Register(zbus.ObjectID{Name: "system", Version: "0.0.1"}, system)
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
	)

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	mon(ctx, server)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}
	}()

	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(redis))
	cap, err := oracle.Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	bus, err := rmb.New(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to initialize message bus server")
	}

	dmi, err := oracle.DMI()
	if err != nil {
		return errors.Wrap(err, "failed to get dmi information")
	}

	bus.WithHandler("zos.system.dmi", func(ctx context.Context, payload []byte) (interface{}, error) {
		return dmi, nil
	})

	// answer calls for dmi
	go func() {
		if err := bus.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("message bus handler failure")
		}
	}()

	node, twin, err := registration(ctx, redis, cap)
	if err != nil {
		return errors.Wrap(err, "failed during node registration")
	}

	log.Info().Uint32("node", node).Uint32("twin", twin).Msg("node registered")

	// uptime update
	go func() {
		for {
			if err := uptime(ctx, redis); err != nil {
				log.Error().Err(err).Msg("sending uptime failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	log.Info().Uint32("twin", twin).Msg("node has been registered")
	log.Debug().Msg("start message bus")

	return runMsgBus(ctx, twin)
}
