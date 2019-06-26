package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/identity"
)

const seedPath = "/var/cache/seed.txt"

func main() {
	nodeID, err := loadIdentify()
	if err != nil {
		os.Exit(1)
	}

	farmID, err := identity.GetFarmID()
	if err != nil {
		log.Error().Err(err).Msg("fail to read farmer id from kernel parameters")
		os.Exit(1)
	}

	store := identity.NewHTTPIDStore("http://172.20.0.1:8080")
	log.Info().Msg("start registration of the node")
	if err := store.RegisterNode(nodeID, farmID); err != nil {
		log.Error().Err(err).Msg("fail to register node identity")
		os.Exit(1)
	}
	log.Info().Msg("node registered successfully")
}

func loadIdentify() (*identity.NodeID, error) {
	if !exists(seedPath) {
		log.Info().Msg("seed not found, generating new key pair")
		keypair, err := identity.GenerateKeyPair()
		if err != nil {
			log.Error().Err(err).Msg("fail to generate key pair for node identity")
			return nil, err
		}

		if err := identity.SerializeSeed(keypair, seedPath); err != nil {
			log.Error().Err(err).Msg("fail to save identity seed on disk")
			return nil, err
		}
	}

	keypair, err := identity.LoadSeed(seedPath)
	if err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	nodeID := identity.NewNodeID(keypair)
	log.Info().
		Str("identify", nodeID.Identity()).
		Msg("node identity loaded")
	return nodeID, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
