package main

import (
	"context"
	"flag"
	"time"

	"github.com/cenkalti/backoff/v3"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gedis"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/version"
)

const module = "monitor"

func cap(ctx context.Context, client zbus.Client) {
	storage := stubs.NewStorageModuleStub(client)
	identity := stubs.NewIdentityManagerStub(client)
	store, err := bcdbClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to bcdb backend")
	}

	r := capacity.NewResourceOracle(storage)

	log.Info().Msg("inspect hardware resources")
	resources, err := r.Total()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read resources capacity from hardware")
	}
	log.Info().
		Uint64("CRU", resources.CRU).
		Uint64("MRU", resources.MRU).
		Uint64("SRU", resources.SRU).
		Uint64("HRU", resources.HRU).
		Msg("resource units found")

	log.Info().Msg("read DMI info")
	dmi, err := r.DMI()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read DMI information from hardware")
	}

	disks, err := r.Disks()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read smartctl information from disks")
	}

	hypervisor, err := r.GetHypervisor()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read virtualized state")
	}

	log.Info().Msg("sends capacity detail to BCDB")
	if err := store.Register(identity.NodeID(), *resources, *dmi, disks, hypervisor); err != nil {
		log.Fatal().Err(err).Msgf("failed to write resources capacity on BCDB")
	}

	sendUptime := func() error {
		uptime, err := r.Uptime()
		if err != nil {
			log.Error().Err(err).Msgf("failed to read uptime")
			return err
		}

		log.Info().Msg("send heart-beat to BCDB")
		if err := store.Ping(identity.NodeID(), uptime); err != nil {
			log.Error().Err(err).Msgf("failed to send heart-beat to BCDB")
			return err
		}
		return nil
	}
	if err := sendUptime(); err != nil {
		log.Fatal().Err(err).Send()
	}

	tick := time.NewTicker(time.Minute * 10)

	go func() {
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				backoff.Retry(sendUptime, backoff.NewExponentialBackOff())
			case <-ctx.Done():
				return
			}
		}
	}()
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

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	cap(ctx, redis)
	mon(ctx, server)

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}

// instantiate the proper client based on the running mode
func bcdbClient() (capacity.Store, error) {
	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse node environment")
	}

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return capacity.NewHTTPStore(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}

	return capacity.NewBCDBStore(store), nil
}
