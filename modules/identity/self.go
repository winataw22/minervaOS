package identity

import (
	"github.com/rs/zerolog/log"
)

const seedPath = "/var/cache/seed.txt"

// LocalNodeID loads the seed use to identify the node itself
func LocalNodeID() (Identifier, error) {
	keypair, err := LoadSeed(seedPath)
	if err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	nodeID := NewNodeID(keypair)
	log.Info().
		Str("identify", nodeID.Identity()).
		Msg("node identity loaded")
	return nodeID, nil
}
