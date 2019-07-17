package identity

import (
	"fmt"
	"io/ioutil"

	"github.com/jbenet/go-base58"

	"golang.org/x/crypto/ed25519"
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

// SerializeSeed saves the seed of a key pair in a file located at path
func SerializeSeed(keypair KeyPair, path string) error {
	seed := keypair.PrivateKey.Seed()

	return ioutil.WriteFile(path, seed, 0400)
}

// LoadSeed reads a seed from a file located at path and re-create a
// KeyPair using the seed
func LoadSeed(path string) (k KeyPair, err error) {
	seed, err := ioutil.ReadFile(path)
	if err != nil {
		return k, err
	}

	if len(seed) != ed25519.SeedSize {
		return k, fmt.Errorf("seed has the wrong size %d", len(seed))
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, privateKey[32:])
	k = KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return k, nil
}
