package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/threefoldtech/zosv2/modules/version"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		msgBrokerCon string
		resURL       string
		tnodbURL     string
		debug        bool
		ver          bool
		lruMaxSize   int
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.StringVar(&resURL, "url", "https://tnodb.dev.grid.tf", "URL of the reservation server to poll from")
	flag.IntVar(&lruMaxSize, "cache", 10, "Number of reservation ID to keep in cache for owenership verification")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if debug {
		log.Logger.Level(zerolog.DebugLevel)
	}

	nodeID, err := identity.LocalNodeID()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load node identity")
	}

	// create context and add middlewares
	ctx := context.Background()
	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	cache := provision.NewCache(lruMaxSize, tnodbURL)

	ctx = provision.WithZBus(ctx, client)
	ctx = provision.WithTnoDB(ctx, tnodbURL)
	ctx = provision.WithCache(ctx, cache)

	// bootstrap:
	// we get all the reservations for this node once
	// and try to deploy what is still valid
	log.Info().Msg("start bootstrap provisioning engine")
	store := provision.NewhHTTPStore(resURL)
	provision.New(&bootstrapSource{
		nodeID: nodeID,
		store:  store,
	}).Run(ctx)

	// From here we start the real provision engine that will live
	// for the rest of the life of the node
	pipe, err := provision.FifoSource("/var/run/reservation.pipe")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to allocation reservation pipe")
	}

	source := pipe
	if len(resURL) != 0 {
		source = provision.CompinedSource(
			pipe,
			provision.HTTPSource(store, nodeID),
		)
	}

	engine := provision.New(source)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}

// bootstrapSource implements a provision.ReservationSource
// that sends all the reservation for this node once
// this is used to boostrap all the workload after a boot
type bootstrapSource struct {
	nodeID identity.Identifier
	store  provision.ReservationStore
}

func (s *bootstrapSource) Reservations(ctx context.Context) <-chan provision.Reservation {
	ch := make(chan provision.Reservation)
	go func() {
		defer close(ch)

		res, err := s.store.Poll(s.nodeID, true)
		if err != nil {
			log.Error().Err(err).Msg("failed to get reservations")
			return
		}
		log.Debug().Msgf("reservations already existing %v", res)
		for _, r := range res {
			ch <- *r
		}
	}()
	return ch
}
