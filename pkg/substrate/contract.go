package substrate

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// ContractState enum
type ContractState struct {
	IsCreated    bool
	IsDeleted    bool
	IsOutOfFunds bool
}

// Decode implementation for the enum type
func (r *ContractState) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		r.IsCreated = true
	case 1:
		r.IsDeleted = true
	case 2:
		r.IsOutOfFunds = true
	default:
		return fmt.Errorf("unknown CertificateType value")
	}

	return nil
}

// Encode implementation
func (r ContractState) Encode(encoder scale.Encoder) (err error) {
	if r.IsCreated {
		err = encoder.PushByte(0)
	} else if r.IsDeleted {
		err = encoder.PushByte(1)
	} else if r.IsOutOfFunds {
		err = encoder.PushByte(2)
	}

	return
}

// Contract structure
type Contract struct {
	Versioned
	ContractID     types.U64
	TwinID         types.U32
	Node           types.U32
	DeploymentData []byte
	DeploymentHash string
	PublicIPsCount types.U32
	State          ContractState
	PublicIPs      []PublicIP
}

// CreateContract creates a contract for deployment
func (s *Substrate) CreateContract(identity *Identity, node uint32, body []byte, hash string, publicIPs uint32) (uint64, error) {
	c, err := types.NewCall(s.meta, "SmartContractModule.create_contract",
		node, body, hash, publicIPs,
	)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	blockHash, err := s.call(identity, c)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create contract")
	}

	if err := s.checkForError(blockHash, types.NewAccountID(identity.PublicKey)); err != nil {
		return 0, err
	}

	return s.GetContractWithHash(node, hash)
}

// UpdateContract updates existing contract
func (s *Substrate) UpdateContract(identity *Identity, contract uint64, body []byte, hash string) (uint64, error) {
	c, err := types.NewCall(s.meta, "SmartContractModule.update_contract",
		contract, body, hash,
	)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	blockHash, err := s.call(identity, c)
	if err != nil {
		return 0, errors.Wrap(err, "failed to update contract")
	}

	if err := s.checkForError(blockHash, types.NewAccountID(identity.PublicKey)); err != nil {
		return 0, err
	}

	return contract, nil
}

// CancelContract creates a contract for deployment
func (s *Substrate) CancelContract(identity *Identity, contract uint64) error {
	c, err := types.NewCall(s.meta, "SmartContractModule.cancel_contract", contract)

	if err != nil {
		return errors.Wrap(err, "failed to cancel call")
	}

	blockHash, err := s.call(identity, c)
	if err != nil {
		return errors.Wrap(err, "failed to cancel contract")
	}

	if err := s.checkForError(blockHash, types.NewAccountID(identity.PublicKey)); err != nil {
		return err
	}

	return nil
}

// GetContract we should not have calls to create contract, instead only get
func (s *Substrate) GetContract(id uint64) (*Contract, error) {
	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}

	key, err := types.CreateStorageKey(s.meta, "SmartContractModule", "Contracts", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	return s.getContract(key)
}

// GetContractWithHash gets a contract given the node id and hash
func (s *Substrate) GetContractWithHash(node uint32, hash string) (uint64, error) {
	nodeBytes, err := types.EncodeToBytes(node)
	if err != nil {
		return 0, err
	}
	hashBytes, err := types.EncodeToBytes(hash)
	if err != nil {
		return 0, err
	}
	key, err := types.CreateStorageKey(s.meta, "SmartContractModule", "ContractIDByNodeIDAndHash", nodeBytes, hashBytes, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create substrate query key")
	}
	var contract types.U64
	_, err = s.cl.RPC.State.GetStorageLatest(key, &contract)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lookup contracts")
	}

	if contract == 0 {
		return 0, errors.Wrap(ErrNotFound, "contract not found")
	}

	return uint64(contract), nil
}

// GetNodeContracts gets all contracts on a node (pk) in given state
func (s *Substrate) GetNodeContracts(node uint32, state ContractState) ([]Contract, error) {
	nodeBytes, err := types.EncodeToBytes(node)
	if err != nil {
		return nil, err
	}
	stateBytes, err := types.EncodeToBytes(state)
	if err != nil {
		return nil, err
	}
	key, err := types.CreateStorageKey(s.meta, "SmartContractModule", "NodeContracts", nodeBytes, stateBytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}
	var contracts []Contract
	_, err = s.cl.RPC.State.GetStorageLatest(key, &contracts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup contracts")
	}

	return contracts, nil
}

func (s *Substrate) getContract(key types.StorageKey) (*Contract, error) {
	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup contract")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "contract not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var node Contract

	switch version {
	case 1:
		if err := types.DecodeFromBytes(*raw, &node); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &node, nil
}

// Consumption structure
type Consumption struct {
	ContractID types.U64
	Timestamp  types.U64
	CRU        types.U64 `json:"cru"`
	SRU        types.U64 `json:"sru"`
	HRU        types.U64 `json:"hru"`
	MRU        types.U64 `json:"mru"`
	NRU        types.U64 `json:"nru"`
}

// IsEmpty true if consumption is zero
func (s *Consumption) IsEmpty() bool {
	//Unit = gridtypes.Megabyte
	return s.CRU == 0 && s.SRU == 0 && s.HRU == 0 && s.MRU == 0
}

// Report send a capacity report to substrate
func (s *Substrate) Report(identity *Identity, consumptions []Consumption) error {
	c, err := types.NewCall(s.meta, "SmartContractModule.add_reports", consumptions)
	if err != nil {
		return errors.Wrap(err, "failed to create call")
	}

	if _, err := s.call(identity, c); err != nil {
		return errors.Wrap(err, "failed to create report")
	}

	return nil
}
