package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/threefoldtech/zosv2/modules/version"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		msgBrokerCon string
		resURL       string
		tnodbURL     string
		storageDir   string
		debug        bool
		ver          bool
	)

	flag.StringVar(&storageDir, "root", "/var/cache/modules/provisiond", "root path of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.StringVar(&resURL, "url", "https://tnodb.dev.grid.tf", "URL of the reservation server to poll from")
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

	if resURL == "" {
		log.Fatal().Msg("reservation URL cannot be empty")
	}

	if err := os.MkdirAll(storageDir, 0770); err != nil {
		log.Fatal().Err(err).Msg("failed to create cache directory")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	// to get reservation from tnodb
	remoteStore := provision.NewHTTPStore(resURL)
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
	ctx = provision.WithTnoDB(ctx, tnodbURL)
	ctx = provision.WithOwnerCache(ctx, ownerCache)
	ctx = provision.WithZDBMapping(ctx, &provision.ZDBMapping{})

	// From here we start the real provision engine that will live
	// for the rest of the life of the node
	source := provision.CombinedSource(
		provision.HTTPSource(remoteStore, nodeID),
		provision.NewDecommissionSource(localStore),
	)

	engine := provision.New(source, localStore)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
