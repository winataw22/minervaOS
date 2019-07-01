package identity

import (
	"io/ioutil"

	"golang.org/x/crypto/ed25519"
)

// KeyPair holds a public and private side of an ed25519 key pair
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKeyPair creates a new KeyPair from a random seed
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	keypair := KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return &keypair, nil
}

// SerializeSeed saves the seed of a key pair in a file located at path
func SerializeSeed(keypair *KeyPair, path string) error {
	seed := keypair.PrivateKey.Seed()

	return ioutil.WriteFile(path, seed, 0400)
}

// LoadSeed reads a seed from a file located at path and re-create a
// KeyPair using the seed
func LoadSeed(path string) (*KeyPair, error) {
	seed, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, privateKey[32:])
	keypair := KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return &keypair, nil
}
