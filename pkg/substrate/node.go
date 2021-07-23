package substrate

import (
	"crypto/ed25519"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// Resources type
type Resources struct {
	HRU types.U64
	SRU types.U64
	CRU types.U64
	MRU types.U64
}

// Location type
type Location struct {
	Longitude string
	Latitude  string
}

// Role type
type Role struct {
	IsNode    bool
	IsGateway bool
}

// Decode implementation for the enum type
func (r *Role) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		r.IsNode = true
	case 1:
		r.IsGateway = true
	default:
		return fmt.Errorf("unknown CertificateType value")
	}

	return nil
}

// Encode implementation
func (r Role) Encode(encoder scale.Encoder) (err error) {
	if r.IsNode {
		err = encoder.PushByte(0)
	} else if r.IsGateway {
		err = encoder.PushByte(1)
	}

	return
}

// PublicConfig type
type PublicConfig struct {
	IPv4 string
	IPv6 string
	GWv4 string
	GWv6 string
}

// OptionPublicConfig type
type OptionPublicConfig struct {
	HasValue bool
	AsValue  PublicConfig
}

// Encode implementation
func (m OptionPublicConfig) Encode(encoder scale.Encoder) (err error) {
	var i byte
	if m.HasValue {
		i = 1
	}
	err = encoder.PushByte(i)
	if err != nil {
		return err
	}

	if m.HasValue {
		err = encoder.Encode(m.AsValue)
	}

	return
}

// Node type
type Node struct {
	Versioned
	ID           types.U32
	FarmID       types.U32
	TwinID       types.U32
	Resources    Resources
	Location     Location
	CountryID    types.U32
	CityID       types.U32
	PublicConfig OptionPublicConfig
}

//GetNodeByTwinID gets a node by twin id
func (s *Substrate) GetNodeByTwinID(twin uint32) (uint32, error) {
	bytes, err := types.EncodeToBytes(twin)
	if err != nil {
		return 0, err
	}
	key, err := types.CreateStorageKey(s.meta, "TfgridModule", "NodeIdByTwinID", bytes, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create substrate query key")
	}
	var id types.U32
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lookup entity")
	}

	if !ok || id == 0 {
		return 0, errors.Wrap(ErrNotFound, "node not found")
	}

	return uint32(id), nil
}

// GetNode with id
func (s *Substrate) GetNode(id uint32) (*Node, error) {
	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(s.meta, "TfgridModule", "Nodes", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	return s.getNode(key)
}

func (s *Substrate) getNode(key types.StorageKey) (*Node, error) {
	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "node not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var node Node

	switch version {
	case 0:
		fallthrough
	case 1:
		if err := types.DecodeFromBytes(*raw, &node); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &node, nil
}

// CreateNode creates a node
func (s *Substrate) CreateNode(sk ed25519.PrivateKey, node Node) (uint32, error) {
	if node.TwinID == 0 {
		return 0, fmt.Errorf("twin id is required")
	}

	c, err := types.NewCall(s.meta, "TfgridModule.create_node",
		node.FarmID, node.Resources, node.Location,
		node.CountryID, node.CityID, node.PublicConfig,
	)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	if err := s.call(sk, c); err != nil {
		return 0, errors.Wrap(err, "failed to create node")
	}

	return s.GetNodeByTwinID(uint32(node.TwinID))

}

// UpdateNode updates a node
func (s *Substrate) UpdateNode(sk ed25519.PrivateKey, node Node) (uint32, error) {
	if node.TwinID == 0 {
		return 0, fmt.Errorf("twin id is required")
	}

	c, err := types.NewCall(s.meta, "TfgridModule.update_node", node.ID, node.FarmID, node.Resources, node.Location,
		node.CountryID, node.CityID, node.PublicConfig,
	)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	if err := s.call(sk, c); err != nil {
		return 0, errors.Wrap(err, "failed to update node")
	}

	return s.GetNodeByTwinID(uint32(node.TwinID))
}
