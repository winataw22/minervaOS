package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/utils"
	"github.com/threefoldtech/zosv2/modules/version"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"

	"github.com/cenkalti/backoff/v3"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/threefoldtech/zosv2/modules/network/types"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

func main() {
	var (
		tnodbURL string
		root     string
		broker   string
		ver      bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/networkd", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if err := network.DefaultBridgeValid(); err != nil {
		log.Fatal().Err(err).Msg("invalid setup")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus broker")
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Error().Err(err).Msgf("fail to create module root")
	}

	db := tnodb.NewHTTPTNoDB(tnodbURL)

	identity := stubs.NewIdentityManagerStub(client)
	networker := network.NewNetworker(identity, db, root)
	nodeID := identity.NodeID()

	if err := publishIfaces(nodeID, db); err != nil {
		log.Error().Err(err).Msg("failed to publish network interfaces to tnodb")
		os.Exit(1)
	}

	ifaceVersion := -1
	exitIface, err := db.ReadPubIface(nodeID)
	if err == nil {
		if err := configurePubIface(exitIface); err != nil {
			log.Error().Err(err).Msg("failed to configure public interface")
			os.Exit(1)
		}
		ifaceVersion = exitIface.Version
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chIface := watchPubIface(ctx, nodeID, db, ifaceVersion)
	go func(ctx context.Context, ch <-chan *types.PubIface) {
		for {
			select {
			case iface := <-ch:
				_ = configurePubIface(iface)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, chIface)

	if err := startServer(ctx, broker, networker); err != nil {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}

func publishIfaces(id modules.Identifier, db network.TNoDB) error {
	f := func() error {
		log.Info().Msg("try to publish interfaces to TNoDB")
		return db.PublishInterfaces(id)
	}
	errHandler := func(err error, _ time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to publish the node interaces")
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}

func startServer(ctx context.Context, broker string, networker modules.Networker) error {

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Error().Err(err).Msgf("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}
