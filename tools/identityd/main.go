package main

import (
	"flag"
	"os"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/identity"
)

const seedPath = "/var/cache/seed.txt"

func main() {
	var (
		tnodbURL string
	)

	flag.StringVar(&tnodbURL, "tnodb", "http://172.20.0.1:8080", "address of tenant network object database")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	nodeID, err := loadIdentity()
	if err != nil {
		os.Exit(1)
	}

	farmID, err := identity.GetFarmID()
	if err != nil {
		log.Error().Err(err).Msg("fail to read farmer id from kernel parameters")
		os.Exit(1)
	}

	store := identity.NewHTTPIDStore(tnodbURL)
	f := func() error {
		log.Info().Msg("start registration of the node")
		if err := store.RegisterNode(nodeID, farmID); err != nil {
			log.Error().Err(err).Msg("fail to register node identity")
			return err
		}
		return nil
	}

	err = backoff.Retry(f, backoff.NewExponentialBackOff())
	if err != nil {
		return
	}

	log.Info().Msg("node registered successfully")
}

func loadIdentity() (identity.Identifier, error) {
	if !exists(seedPath) {
		log.Info().Msg("seed not found, generating new key pair")
		nodeID, err := identity.GenerateKeyPair()
		if err != nil {
			log.Error().Err(err).Msg("fail to generate key pair for node identity")
			return nil, err
		}

		if err := identity.SerializeSeed(nodeID, seedPath); err != nil {
			log.Error().Err(err).Msg("fail to save identity seed on disk")
			return nil, err
		}
	}

	nodeID, err := identity.LoadSeed(seedPath)
	if err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	log.Info().
		Str("identify", nodeID.Identity()).
		Msg("node identity loaded")
	return nodeID, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
