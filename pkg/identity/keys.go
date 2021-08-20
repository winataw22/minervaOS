package identity

import (
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg/versioned"
	"github.com/tyler-smith/go-bip39"

	"golang.org/x/crypto/ed25519"
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

// KeyPair holds a public and private side of an ed25519 key pair
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// Identity implements the Identifier interface
func (k KeyPair) Identity() string {
	return base58.Encode(k.PublicKey)
}

// GenerateKeyPair creates a new KeyPair from a random seed
func GenerateKeyPair() (k KeyPair, err error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return k, err
	}
	k = KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return k, nil
}

// Save saves the seed of a key pair in a file located at path
func (k *KeyPair) Save(path string) error {
	seed := k.PrivateKey.Seed()

	return versioned.WriteFile(path, SeedVersion1, seed, 0400)
}

// LoadSeed from path
func LoadSeed(path string) ([]byte, error) {
	version, seed, err := versioned.ReadFile(path)
	if versioned.IsNotVersioned(err) {
		// this is a compatibility code for seed files
		// in case it does not have any version information
		if err := versioned.WriteFile(path, SeedVersionLatest, seed, 0400); err != nil {
			return nil, err
		}
		version = SeedVersion1
	} else if err != nil {
		return nil, err
	}

	if version.NE(SeedVersion1) && version.NE(SeedVersion11) {
		return nil, fmt.Errorf("unknown seed version")
	}

	if version.EQ(SeedVersion1) {
		return seed, nil
	}
	// it means we read json data instead of the secret
	type Seed110Struct struct {
		Mnemonics string `json:"mnemonic"`
	}
	var seed110 Seed110Struct
	if err = json.Unmarshal(seed, &seed110); err != nil {
		return nil, err
	}
	return bip39.EntropyFromMnemonic(seed110.Mnemonics)
}

// LoadKeyPair reads a seed from a file located at path and re-create a
// KeyPair using the seed
func LoadKeyPair(path string) (k KeyPair, err error) {
	return loadKeyPair(path)
}

// LoadLegacyKeyPair load keypair without deprecated message for converted
func LoadLegacyKeyPair(path string) (k KeyPair, err error) {
	return loadKeyPair(path)
}

func loadKeyPair(path string) (k KeyPair, err error) {
	seed, err := LoadSeed(path)
	if err != nil {
		return k, err
	}

	return FromSeed(seed)
}

// FromSeed creates a new key pair from seed
func FromSeed(seed []byte) (pair KeyPair, err error) {
	if len(seed) != ed25519.SeedSize {
		return pair, fmt.Errorf("seed has the wrong size %d and should be %d", len(seed), ed25519.SeedSize)
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, privateKey[32:])
	pair = KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	return pair, nil
}
