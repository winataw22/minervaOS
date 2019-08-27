package identity

import (
	"fmt"
	"os"

	"github.com/threefoldtech/zosv2/modules/crypto"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/kernel"
)

type identityManager struct {
	key KeyPair
}

// NewManager creates an identity daemon from seed
// The daemon will auto generate a new seed if the path does
// not exist
func NewManager(path string) (modules.IdentityManager, error) {
	var pair KeyPair
	if seed, err := LoadSeed(path); os.IsNotExist(err) {
		pair, err = GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate key pair")
		}

		if err := pair.Save(path); err != nil {
			return nil, errors.Wrap(err, "failed to persist key seed")
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to load seed")
	} else {
		pair, err = FromSeed(seed)
		if err != nil {
			return nil, errors.Wrap(err, "invalid seed file")
		}
	}

	return &identityManager{pair}, nil
}

// NodeID returns the node identity
func (d *identityManager) NodeID() modules.StrIdentifier {
	return modules.StrIdentifier(d.key.Identity())
}

// FarmID returns the farm ID of the node or an error if no farm ID is configured
func (d *identityManager) FarmID() (modules.StrIdentifier, error) {
	params := kernel.GetParams()

	farmerID, found := params.Get("farmer_id")
	if !found {
		return "", fmt.Errorf("farmer id not found in kernel parameters")
	}

	return modules.StrIdentifier(farmerID[0]), nil
}

// Sign signs the message with privateKey and returns a signature.
func (d *identityManager) Sign(message []byte) ([]byte, error) {
	return crypto.Sign(d.key.PrivateKey, message)
}

// Verify reports whether sig is a valid signature of message by publicKey.
func (d *identityManager) Verify(message, sig []byte) error {
	return crypto.Verify(d.key.PublicKey, message, sig)
}

// Encrypt encrypts message with the public key of the node
func (d *identityManager) Encrypt(message []byte) ([]byte, error) {
	return crypto.Encrypt(message, d.key.PublicKey)
}

// Decrypt decrypts message with the private of the node
func (d *identityManager) Decrypt(message []byte) ([]byte, error) {
	return crypto.Decrypt(message, d.key.PrivateKey)
}
