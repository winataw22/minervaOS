package identity

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/kernel"
	"golang.org/x/crypto/ed25519"
)

const (
	// SeedPath default seed path
	SeedPath = "/var/cache/seed.txt"
)

type identityManager struct {
	key KeyPair
}

// NewManager creates an identity daemon from seed
// The daemon will auto generate a new seed if the path does
// not exist
func NewManager(path string) (modules.IdentityManager, error) {
	var pair KeyPair
	if seed, err := ioutil.ReadFile(path); os.IsNotExist(err) {
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
			return nil, errors.Wrap(err, "invlaid seed file")
		}
	}

	return &identityManager{pair}, nil
}

// LocalNodeID loads the seed use to identify the node itself
func (d *identityManager) NodeID() modules.StrIdentifier {
	return modules.StrIdentifier(d.key.Identity())
}

func (d *identityManager) FarmID() (modules.StrIdentifier, error) {
	params := kernel.GetParams()

	farmerID, found := params.Get("farmer_id")
	if !found {
		return "", fmt.Errorf("farmer id not found in kernel parameters")
	}

	return modules.StrIdentifier(farmerID[0]), nil
}

// Sign signs the message with privateKey and returns a signature.
func (d *identityManager) Sign(data []byte) ([]byte, error) {
	if len(d.key.PrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key has the wrong size")
	}
	return ed25519.Sign(d.key.PrivateKey, data), nil
}

// Verify reports whether sig is a valid signature of message by publicKey.
func (d *identityManager) Verify(data, sig []byte) error {
	if len(d.key.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("public key has the wrong size")
	}

	if !ed25519.Verify(d.key.PublicKey, data, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}
