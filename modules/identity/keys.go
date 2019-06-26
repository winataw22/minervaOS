package identity

import (
	"io/ioutil"

	"golang.org/x/crypto/ed25519"
)

type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

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

func SerializeSeed(keypair *KeyPair, path string) error {
	seed := keypair.PrivateKey.Seed()

	return ioutil.WriteFile(path, seed, 0400)
}

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
