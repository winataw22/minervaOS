package identity

import (
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zosv2/modules/versioned"

	"golang.org/x/crypto/ed25519"
)

var (
	//SeedVersion1 version
	seedVersion1 = versioned.MustParse("1.0.0")
	//SeedVersionLatest link to latest seed version
	seedVersionLatest = seedVersion1
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

	return versioned.WriteFile(path, seedVersionLatest, seed, 0400)
}

// LoadSeed from path
func LoadSeed(path string) ([]byte, error) {
	version, seed, err := versioned.ReadFile(path)
	if versioned.IsNotVersioned(err) {
		// this is a compatibility code for seed files
		// in case it does not have any version information
		versioned.WriteFile(path, seedVersionLatest, seed, 0400)
		version = seedVersionLatest
	} else if err != nil {
		return nil, err
	}

	if version.NE(seedVersionLatest) {
		return nil, fmt.Errorf("unknown seed version")
	}

	return seed, nil
}

// LoadKeyPair reads a seed from a file located at path and re-create a
// KeyPair using the seed
func LoadKeyPair(path string) (k KeyPair, err error) {
	seed, err := LoadSeed(path)
	if err != nil {
		return k, err
	}

	return FromSeed(seed)
}

// FromSeed creates a new key pair from seed
func FromSeed(seed []byte) (pair KeyPair, err error) {
	if len(seed) != ed25519.SeedSize {
		return pair, fmt.Errorf("seed has the wrong size %d", len(seed))
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
