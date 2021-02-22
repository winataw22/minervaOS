package substrate

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v2/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/pkg/errors"
)

// CertificationType is a substrate enum
type CertificationType struct {
	IsNone   bool
	IsSilver bool
	IsGold   bool
}

// Decode implementation for the enum type
func (p *CertificationType) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsNone = true
	case 1:
		p.IsSilver = true
	case 2:
		p.IsGold = true
	default:
		return fmt.Errorf("unknown CertificateType value")
	}

	return nil
}

// Farm type
type Farm struct {
	Versioned
	ID                types.U32
	Name              string
	TwinID            types.U32
	PricingPolicyID   types.U32
	CertificationType CertificationType
	CountryID         types.U32
	CityID            types.U32
}

func (s *substrateClient) GetFarm(id uint32) (*Farm, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Farms", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "farm not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var farm Farm

	switch version {
	case 1:
		if err := types.DecodeFromBytes(*raw, &farm); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &farm, nil
}
