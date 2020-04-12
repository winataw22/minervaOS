package main

import (
	"context"
	"flag"
	"time"

	"github.com/cenkalti/backoff/v3"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/version"
)

const module = "monitor"

func cap(ctx context.Context, client zbus.Client) {
	storage := stubs.NewStorageModuleStub(client)
	identity := stubs.NewIdentityManagerStub(client)
	cl, err := bcdbClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to bcdb backend")
	}

	// call this now so we block here until identityd is ready to serve us
	nodeID := identity.NodeID().Identity()

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

	ru := directory.ResourceAmount{
		Cru: resources.CRU,
		Mru: float64(resources.MRU),
		Hru: float64(resources.HRU),
		Sru: float64(resources.SRU),
	}

	setCapacity := func() error {
		log.Info().Msg("sends capacity detail to BCDB")
		return cl.NodeSetCapacity(nodeID, ru, *dmi, disks, hypervisor)
	}
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // retry forever
	backoff.RetryNotify(setCapacity, bo, func(err error, d time.Duration) {
		log.Error().
			Err(err).
			Str("sleep", d.String()).
			Msgf("failed to write resources capacity on BCDB")
	})

	sendUptime := func() error {
		uptime, err := r.Uptime()
		if err != nil {
			log.Error().Err(err).Msgf("failed to read uptime")
			return err
		}

		log.Info().Msg("send heart-beat to BCDB")
		if err := cl.NodeUpdateUptime(identity.NodeID().Identity(), uptime); err != nil {
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
func bcdbClient() (client.Directory, error) {
	client, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	return client.Directory, nil
}
