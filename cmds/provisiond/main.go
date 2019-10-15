package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gedis"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/version"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

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

	if debug {
		log.Logger.Level(zerolog.DebugLevel)
	}

	flag.Parse()

	if err := os.MkdirAll(storageDir, 0770); err != nil {
		log.Fatal().Err(err).Msg("failed to create cache directory")
	}

	env := environment.Get()

	if env.Orphan {
		// disable providiond on this node
		// we don't have a valid farmer id set
		log.Fatal().Msg("orphan node, we won't provision anything at all")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	// to get reservation from tnodb
	remoteStore, err := bcdbClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to instantiate BCDB client")
	}
	// to store reservation locally on the node
	localStore, err := provision.NewFSStore(filepath.Join(storageDir, "reservations"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create local reservation store")
	}
	// to get the user ID of a reservation
	ownerCache := provision.NewCache(localStore, remoteStore)

	// create context and add middlewares
	ctx := context.Background()
	ctx = provision.WithZBus(ctx, client)
	ctx = provision.WithOwnerCache(ctx, ownerCache)
	ctx = provision.WithZDBMapping(ctx, provision.NewZDBMapping())

	// From here we start the real provision engine that will live
	// for the rest of the life of the node
	source := provision.CombinedSource(
		provision.PollSource(remoteStore, nodeID),
		provision.NewDecommissionSource(localStore),
	)

	engine := provision.New(source, localStore, remoteStore)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}

type store interface {
	provision.ReservationGetter
	provision.ReservationPoller
	provision.Feedbacker
}

// instantiate the proper client based on the running mode
func bcdbClient() (store, error) {
	env := environment.Get()

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return provision.NewHTTPStore(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbNamespace, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}
	return store, nil
}
