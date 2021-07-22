package substrate

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// CertificationType is a substrate enum
type CertificationType struct {
	IsDiy       bool
	IsCertified bool
}

// Decode implementation for the enum type
func (p *CertificationType) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsDiy = true
	case 1:
		p.IsCertified = true
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
	PublicIPs         []PublicIP
}

// PublicIP structure
type PublicIP struct {
	IP         string
	Gateway    string
	ContractID types.U64
}

// GetFarm gets a farm with ID
func (s *Substrate) GetFarm(id uint32) (*Farm, error) {
	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(s.meta, "TfgridModule", "Farms", bytes, nil)
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
