package identity

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/identity/store"
)

const (
	seedName = "seed.txt"
	// disableTpm support completely for now
	// until all tests (PRC changes) are covered
	disableTpm = true
)

// NewStore tries to build the best key store available
// for this ndoe.
// On a machine with no tpm support, that would be a file
// store.
// If TPM is supported, TPM will be used.
// There is a special case if tpm is supported, but a file seed
// exits, this file key will be migrated to the TPM store then
// deleted (only if delete is set to true)
func NewStore(root string, delete bool) (store.Store, error) {
	file := store.NewFileStore(filepath.Join(root, seedName))
	if disableTpm || !store.IsTPMEnabled() {
		return file, nil
	}

	// tpm is supported, but do we have a key
	tpm := store.NewTPM()
	exists, err := file.Exists()
	if err != nil {
		return nil, fmt.Errorf("failed to check for seed file: %s", err)
	}

	if !exists {
		return tpm, nil
	}

	if ok, err := tpm.Exists(); err == nil && ok {
		// so there is a key on disk, but tpm already has a stored key
		// then we still just return no need for migration to avoid
		// overriding the key in tpm
		return tpm, nil
	}

	// if we failed to get the key from store
	// may be better generate a new one?
	// todo: need discussion

	key, err := file.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load key from file: %w", err)
	}

	// migration of key
	if err := tpm.Set(key); err != nil {
		// we failed to do migration but we have a valid key.
		// we shouldn't then fail instead use the file store
		log.Error().Err(err).Msg("failed to migrate key to tpm store")
		return file, nil
	}

	if delete {
		if err := file.Annihilate(); err != nil {
			log.Error().Err(err).Msg("failed to clear up key file")
		}
	}

	return tpm, nil
}
