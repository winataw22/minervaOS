package identity

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/identity/store"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
)

type identityManager struct {
	kind string
	key  KeyPair
	sub  substrate.Manager
	env  environment.Environment

	farm string
}

// NewManager creates an identity daemon from seed
// The daemon will auto generate a new seed if the path does
// not exist
// debug flag is used to change the behavior slightly when zos is running in debug
// mode. Right now only the key store uses this flag. In case of debug migrated keys
// to tpm are not deleted from disks. This allow switching back and forth between tpm
// and non-tpm key stores.
func NewManager(root string, debug bool) (pkg.IdentityManager, error) {
	st, err := NewStore(root, !debug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create key store")
	}
	log.Info().Str("kind", st.Kind()).Msg("key store loaded")
	key, err := st.Get()
	var pair KeyPair
	if errors.Is(err, store.ErrKeyDoesNotExist) {
		pair, err = GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate key pair")
		}
		if err := st.Set(pair.PrivateKey); err != nil {
			return nil, errors.Wrap(err, "failed to persist key seed")
		}
	} else if err != nil {
		log.Error().Err(err).Msg("failed to load key. to recover the key data will be deleted and regenerated")
		if err := st.Annihilate(); err != nil {
			log.Error().Err(err).Msg("failed to clean up key store")
		}
		return nil, errors.Wrap(err, "failed to load seed")
	} else {
		pair = KeyPairFromKey(key)
	}

	sub, err := environment.GetSubstrate()
	if err != nil {
		return nil, err
	}
	env, err := environment.Get()
	if err != nil {
		return nil, err
	}

	return &identityManager{
		kind: st.Kind(),
		key:  pair,
		sub:  sub,
		env:  env,
	}, nil
}

// StoreKind returns store kind
func (d *identityManager) StoreKind() string {
	return d.kind
}

// NodeID returns the node identity
func (d *identityManager) NodeID() pkg.StrIdentifier {
	return pkg.StrIdentifier(d.key.Identity())
}

// NodeID returns the node identity
func (d *identityManager) Address() (pkg.Address, error) {
	id, err := substrate.NewIdentityFromEd25519Key(d.key.PrivateKey)
	if err != nil {
		return "", err
	}
	return pkg.Address(id.Address()), nil
}

func (d *identityManager) Farm() (string, error) {
	if len(d.farm) != 0 {
		return d.farm, nil
	}

	cl, err := d.sub.Substrate()
	if err != nil {
		return "", err
	}
	defer cl.Close()

	farm, err := cl.GetFarm(uint32(d.env.FarmerID))
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
