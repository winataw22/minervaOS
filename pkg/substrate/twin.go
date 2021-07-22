package substrate

import (
	"crypto/ed25519"
	"net"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// EntityProof struct
type EntityProof struct {
	EntityID  types.U32
	Signature string
}

// Twin struct
type Twin struct {
	Versioned
	ID       types.U32
	Account  AccountID
	IP       string
	Entities []EntityProof
}

//IPAddress parse the twin IP as net.IP
func (t *Twin) IPAddress() net.IP {
	return net.ParseIP(t.IP)
}

// GetTwinByPubKey gets a twin with public key
func (s *Substrate) GetTwinByPubKey(pk []byte) (uint32, error) {
	key, err := types.CreateStorageKey(s.meta, "TfgridModule", "TwinIdByAccountID", pk, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create substrate query key")
	}
	var id types.U32
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lookup entity")
	}

	if !ok || id == 0 {
		return 0, errors.Wrap(ErrNotFound, "twin not found")
	}

	return uint32(id), nil
}

// GetTwin gets a twin
func (s *Substrate) GetTwin(id uint32) (*Twin, error) {
	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(s.meta, "TfgridModule", "Twins", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "twin not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var twin Twin

	switch version {
	case 1:
		if err := types.DecodeFromBytes(*raw, &twin); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &twin, nil
}

// CreateTwin creates a twin
func (s *Substrate) CreateTwin(sk ed25519.PrivateKey, ip net.IP) (uint32, error) {
	c, err := types.NewCall(s.meta, "TfgridModule.create_twin", ip.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	if err := s.call(sk, c); err != nil {
		return 0, errors.Wrap(err, "failed to create twin")
	}

	identity, err := Identity(sk)
	if err != nil {
		return 0, err
	}

	return s.GetTwinByPubKey(identity.PublicKey)
}
