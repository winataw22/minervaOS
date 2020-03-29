package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	module = "provision"
)

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		storageDir   string
		debug        bool
		ver          bool
	)

	flag.StringVar(&storageDir, "root", "/var/cache/modules/provisiond", "root path of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// keep checking if limited-cache flag is set
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("failed cache reservation! Retrying every 30 seconds...")
		for app.CheckFlag(app.LimitedCache) {
			time.Sleep(time.Second * 30)
		}
	}

	flag.Parse()

	if err := os.MkdirAll(storageDir, 0770); err != nil {
		log.Fatal().Err(err).Msg("failed to create cache directory")
	}

	env, err := environment.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse node environment")
	}

	if env.Orphan {
		// disable providiond on this node
		// we don't have a valid farmer id set
		log.Fatal().Msg("orphan node, we won't provision anything at all")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to message broker")
	}
	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	// to get reservation from tnodb
	cl, err := app.ExplorerClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to instantiate BCDB client")
	}
	// to store reservation locally on the node
	localStore, err := provision.NewFSStore(filepath.Join(storageDir, "reservations"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create local reservation store")
	}
	// to get the user ID of a reservation
	ownerCache := provision.NewCache(localStore, provision.ReservationGetterFromWorkloads(cl.Workloads))

	// create context and add middlewares
	ctx := context.Background()
	ctx = provision.WithZBus(ctx, client)
	ctx = provision.WithOwnerCache(ctx, ownerCache)

	// From here we start the real provision engine that will live
	// for the rest of the life of the node
	source := provision.CombinedSource(
		provision.PollSource(provision.ReservationPollerFromWorkloads(cl.Workloads), nodeID),
		provision.NewDecommissionSource(localStore),
	)

	engine := provision.New(nodeID.Identity(), source, localStore, cl)

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, pkg.ProvisionMonitor(engine))

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
		log.Info().Msg("zbus server stopped")
	}()

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
	log.Info().Msg("provision engine stopped")
}

type store interface {
	provision.ReservationGetter
	provision.ReservationPoller
	provision.Feedbacker
}
