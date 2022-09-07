/*
Key store implements different methods of storing the node identity seed on disk
*/
package store

import (
	"crypto/ed25519"
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrKeyDoesNotExist = fmt.Errorf("key does not exist")
	ErrInvalidKey      = fmt.Errorf("invalid key data")
)

type Store interface {
	// Get returns the key from the store
	Get() (ed25519.PrivateKey, error)
	// Updates, or overrides the current key
	Set(key ed25519.PrivateKey) error
	// Check if key there is a key stored in the
	// store
	Exists() (bool, error)
	// Destroys the key
	Annihilate() error
	// Kind returns store kind
	Kind() string
}

// keyFromSeed creates a new key pair from seed
func keyFromSeed(seed []byte) (key ed25519.PrivateKey, err error) {
	if len(seed) != ed25519.SeedSize {
		return nil, errors.Wrapf(ErrInvalidKey, "seed has the wrong size %d and should be %d", len(seed), ed25519.SeedSize)
	}

	return ed25519.NewKeyFromSeed(seed), nil
}
