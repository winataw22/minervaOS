package main

import (
	"context"
	"flag"
	"os"

	"github.com/threefoldtech/zbus"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/version"
)

const (
	seedPath = "/var/cache/seed.txt"
	module   = "identityd"
	workers  = 10
)

func main() {
	var (
		msgBrokerCon string
		tnodbURL     string
		ver      bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workers)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create zbus server")
	}

	manager, err := identity.NewManager(seedPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity manager")
	}

	farmID, err := manager.FarmID()
	if err != nil {
		log.Fatal().Err(err).Msg("fail to read farmer id from kernel parameters")
	}

	nodeID := manager.NodeID()
	log.Info().
		Str("identity", nodeID.Identity()).
		Msg("node identity loaded")

	// Node registration can happen in the background.
	go func() {
		store := identity.NewHTTPIDStore(tnodbURL)
		f := func() error {
			log.Info().Msg("start registration of the node")
			if err := store.RegisterNode(nodeID, farmID); err != nil {
				log.Error().Err(err).Msg("fail to register node identity")
				return err
			}
			return nil
		}

		if err := backoff.Retry(f, backoff.NewExponentialBackOff()); err == nil {
			log.Info().Msg("node registered successfully")
		}
	}()

	if err := server.Register(zbus.ObjectID{"manager", "0.0.1"}, manager); err != nil {
		log.Fatal().Err(err).Msg("failed to register identity manager")
	}

	if err := server.Run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("server exit")
	}
}
