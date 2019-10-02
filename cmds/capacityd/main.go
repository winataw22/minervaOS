package main

import (
	"flag"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gedis"
	"github.com/threefoldtech/zos/pkg/stubs"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/version"
)

const module = "capacity"

func main() {
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
	storage := stubs.NewStorageModuleStub(redis)
	identity := stubs.NewIdentityManagerStub(redis)
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

	log.Info().Msg("sends capacity detail to BCDB")
	if err := store.Register(identity.NodeID(), *resources, *dmi); err != nil {
		log.Fatal().Err(err).Msgf("failed to write resources capacity on BCDB")
	}

	for {
		<-time.After(time.Minute * 10)

		log.Info().Msg("send heart-beat to BCDB")
		if err := store.Ping(identity.NodeID()); err != nil {
			log.Error().Err(err).Msgf("failed to send heart-beat to BCDB")
		}
	}
}

// instantiate the proper client based on the running mode
func bcdbClient() (capacity.Store, error) {
	env := environment.Get()

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return capacity.NewHTTPStore(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbNamespace, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}

	return capacity.NewBCDBStore(store), nil
}
