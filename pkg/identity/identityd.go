package identity

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/crypto"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
)

type identityManager struct {
	key KeyPair
	sub *substrate.Substrate
	env environment.Environment

	farm string
	node uint32
}

// NewManager creates an identity daemon from seed
// The daemon will auto generate a new seed if the path does
// not exist
func NewManager(path string) (pkg.IdentityManager, error) {
	env, err := environment.Get()
	if err != nil {
		return nil, err
	}
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
		if err := os.Remove(path); err != nil {
			log.Error().Err(err).Msg("failed to delete corrupt seed file")
		}
		return nil, errors.Wrap(err, "failed to load seed")
	} else {
		pair, err = FromSeed(seed)
		if err != nil {
			return nil, errors.Wrap(err, "invalid seed file")
		}
	}

	sub, err := substrate.NewSubstrate(env.SubstrateURL)
	if err != nil {
		return nil, err
	}
	return &identityManager{
		key: pair,
		sub: sub,
		env: env,
	}, nil
}

// NodeID returns the node identity
func (d *identityManager) NodeID() pkg.StrIdentifier {
	return pkg.StrIdentifier(d.key.Identity())
}

func (d *identityManager) NodeIDNumeric() (uint32, error) {
	if d.node != 0 {
		return d.node, nil
	}
	id, err := substrate.IdentityFromSecureKey(d.key.PrivateKey)
	if err != nil {
		return 0, err
	}
	twin, err := d.sub.GetTwinByPubKey(id.PublicKey)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get node twin")
	}

	node, err := d.sub.GetNodeByTwinID(twin)
	if err != nil {
		return 0, err
	}
	d.node = node
	return node, nil
}

func (d *identityManager) Farm() (string, error) {
	if len(d.farm) != 0 {
		return d.farm, nil
	}

	farm, err := d.sub.GetFarm(uint32(d.env.FarmerID))
	if errors.Is(err, substrate.ErrNotFound) {
		return "", fmt.Errorf("wrong farm id")
	} else if err != nil {
		return "", err
	}

	d.farm = farm.Name
	return farm.Name, nil
}

// FarmID returns the farm ID of the node or an error if no farm ID is configured
func (d *identityManager) FarmID() (pkg.FarmID, error) {
	env, err := environment.Get()
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse node environment")
	}
	return env.FarmerID, nil
}

// FarmSecret returns farm secret from kernel params
func (d *identityManager) FarmSecret() (string, error) {
	env, err := environment.Get()
	if err != nil {
		return "", errors.Wrap(err, "failed to parse node environment")
	}
	return env.FarmSecret, nil
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

// EncryptECDH encrypt msg using AES with shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
// the nonce if prepended to the encrypted message
func (d *identityManager) EncryptECDH(msg []byte, pk []byte) ([]byte, error) {
	return crypto.EncryptECDH(msg, d.key.PrivateKey, pk)
}

// DecryptECDH decrypt AES encrypted msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
func (d *identityManager) DecryptECDH(msg []byte, pk []byte) ([]byte, error) {
	return crypto.DecryptECDH(msg, d.key.PrivateKey, pk)
}

// PrivateKey returns the private key of the node
func (d *identityManager) PrivateKey() []byte {
	return d.key.PrivateKey
}
