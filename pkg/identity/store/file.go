package store

import (
	"crypto/ed25519"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/versioned"
	"github.com/tyler-smith/go-bip39"
)

// Version History:
//   1.0.0: seed binary directly encoded
//   1.1.0: json with key mnemonic and threebot id

var (
	// SeedVersion1 (binary seed)
	SeedVersion1 = versioned.MustParse("1.0.0")
	// SeedVersion11 (json mnemonic)
	SeedVersion11 = versioned.MustParse("1.1.0")
	// SeedVersionLatest link to latest seed version
	SeedVersionLatest = SeedVersion11
)

type FileStore struct {
	path string
}

var _ Store = (*FileStore)(nil)

func NewFileStore(path string) *FileStore {
	return &FileStore{path}
}

func (f *FileStore) Kind() string {
	return "file-store"
}

func (f *FileStore) Set(key ed25519.PrivateKey) error {
	seed := key.Seed()
	return versioned.WriteFile(f.path, SeedVersion1, seed, 0400)
}

func (f *FileStore) Annihilate() error {
	return os.Remove(f.path)
}

func (f *FileStore) Get() (ed25519.PrivateKey, error) {
	version, data, err := versioned.ReadFile(f.path)
	if versioned.IsNotVersioned(err) {
		// this is a compatibility code for seed files
		// in case it does not have any version information
		if err := versioned.WriteFile(f.path, SeedVersionLatest, data, 0400); err != nil {
			return nil, err
		}
		version = SeedVersion1
	} else if os.IsNotExist(err) {
		return nil, ErrKeyDoesNotExist
	} else if err != nil {
		return nil, err
	}

	if version.NE(SeedVersion1) && version.NE(SeedVersion11) {
		return nil, errors.Wrap(ErrInvalidKey, "unknown seed version")
	}

	if version.EQ(SeedVersion1) {
		return keyFromSeed(data)
	}
	// it means we read json data instead of the secret
	type Seed110Struct struct {
		Mnemonics string `json:"mnemonic"`
	}
	var seed110 Seed110Struct
	if err = json.Unmarshal(data, &seed110); err != nil {
		return nil, errors.Wrapf(ErrInvalidKey, "failed to decode seed: %s", err)
	}

	seed, err := bip39.EntropyFromMnemonic(seed110.Mnemonics)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidKey, "failed to decode mnemonics: %s", err)
	}

	return keyFromSeed(seed)
}

func (f *FileStore) Exists() (bool, error) {
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check seed file")
	}

	return true, nil
}
