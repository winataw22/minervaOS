package substrate

import (
	"net"

	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
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

func (s *substrateClient) GetTwin(id uint32) (*Twin, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Twins", bytes, nil)
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
