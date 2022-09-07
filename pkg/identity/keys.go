package identity

import (
	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg/versioned"

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

func KeyPairFromKey(sk ed25519.PrivateKey) KeyPair {
	return KeyPair{
		PrivateKey: sk,
		PublicKey:  sk.Public().(ed25519.PublicKey),
	}
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
