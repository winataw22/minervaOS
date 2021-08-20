package substrate

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vedhavyas/go-subkey"
	subkeyEd25519 "github.com/vedhavyas/go-subkey/ed25519"
	"golang.org/x/crypto/blake2b"
)

// https://github.com/threefoldtech/tfchain_pallets/blob/bc9c5d322463aaf735212e428da4ea32b117dc24/pallet-smart-contract/src/lib.rs#L58
var smartContractModuleErrors = []string{
	"TwinNotExists",
	"NodeNotExists",
	"FarmNotExists",
	"FarmHasNotEnoughPublicIPs",
	"FarmHasNotEnoughPublicIPsFree",
	"FailedToReserveIP",
	"FailedToFreeIPs",
	"ContractNotExists",
	"TwinNotAuthorizedToUpdateContract",
	"TwinNotAuthorizedToCancelContract",
	"NodeNotAuthorizedToDeployContract",
	"NodeNotAuthorizedToComputeReport",
	"PricingPolicyNotExists",
	"ContractIsNotUnique",
	"NameExists",
	"NameNotValid",
}

// TODO: add all events from SmartContractModule and TfgridModule

// ContractCreated is the contract created event
type ContractCreated struct {
	Phase    types.Phase
	Contract Contract
	Topics   []types.Hash
}

// ContractUpdated is the contract updated event
type ContractUpdated struct {
	Phase    types.Phase
	Contract Contract
	Topics   []types.Hash
}

// ContractCanceled is the contract canceled event
type ContractCanceled struct {
	Phase      types.Phase
	ContractID types.U64
	Topics     []types.Hash
}

// EventRecords is a struct that extends the default events with our events
type EventRecords struct {
	types.EventRecords
	SmartContractModule_ContractCreated  []ContractCreated  //nolint:stylecheck,golint
	SmartContractModule_ContractUpdated  []ContractUpdated  //nolint:stylecheck,golint
	SmartContractModule_ContractCanceled []ContractCanceled //nolint:stylecheck,golint
}

// Sign signs data with the private key under the given derivation path, returning the signature. Requires the subkey
// command to be in path
func signBytes(data []byte, privateKeyURI string) ([]byte, error) {
	// if data is longer than 256 bytes, hash it first
	if len(data) > 256 {
		h := blake2b.Sum256(data)
		data = h[:]
	}

	scheme := subkeyEd25519.Scheme{}
	kyr, err := subkey.DeriveKeyPair(scheme, privateKeyURI)
	if err != nil {
		return nil, err
	}

	signature, err := kyr.Sign(data)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// Sign adds a signature to the extrinsic
func (s *Substrate) sign(e *types.Extrinsic, signer *Identity, o types.SignatureOptions) error {
	if e.Type() != types.ExtrinsicVersion4 {
		return fmt.Errorf("unsupported extrinsic version: %v (isSigned: %v, type: %v)", e.Version, e.IsSigned(), e.Type())
	}

	mb, err := types.EncodeToBytes(e.Method)
	if err != nil {
		return err
	}

	era := o.Era
	if !o.Era.IsMortalEra {
		era = types.ExtrinsicEra{IsImmortalEra: true}
	}

	payload := types.ExtrinsicPayloadV4{
		ExtrinsicPayloadV3: types.ExtrinsicPayloadV3{
			Method:      mb,
			Era:         era,
			Nonce:       o.Nonce,
			Tip:         o.Tip,
			SpecVersion: o.SpecVersion,
			GenesisHash: o.GenesisHash,
			BlockHash:   o.BlockHash,
		},
		TransactionVersion: o.TransactionVersion,
	}

	signerPubKey := types.NewMultiAddressFromAccountID(signer.PublicKey)

	b, err := types.EncodeToBytes(payload)
	if err != nil {
		return err
	}

	sig, err := signBytes(b, signer.URI)

	if err != nil {
		return err
	}

	extSig := types.ExtrinsicSignatureV4{
		Signer:    signerPubKey,
		Signature: types.MultiSignature{IsEd25519: true, AsEd25519: types.NewSignature(sig)},
		Era:       era,
		Nonce:     o.Nonce,
		Tip:       o.Tip,
	}

	e.Signature = extSig

	// mark the extrinsic as signed
	e.Version |= types.ExtrinsicBitSigned

	return nil
}

func (s *Substrate) call(identity *Identity, call types.Call) (hash types.Hash, err error) {
	// Create the extrinsic
	ext := types.NewExtrinsic(call)

	genesisHash, err := s.cl.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return hash, errors.Wrap(err, "failed to get genesisHash")
	}

	rv, err := s.cl.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return hash, err
	}

	//node.Address =identity.PublicKey
	account, err := s.getAccount(identity, s.meta)
	if err != nil {
		return hash, errors.Wrap(err, "failed to get account")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(account.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: 1,
	}

	err = s.sign(&ext, identity, o)
	if err != nil {
		return hash, errors.Wrap(err, "failed to sign")
	}

	// Send the extrinsic
	sub, err := s.cl.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return hash, errors.Wrap(err, "failed to submit extrinsic")
	}

	defer sub.Unsubscribe()

	for event := range sub.Chan() {
		if event.IsFinalized {
			hash = event.AsFinalized
			break
		} else if event.IsDropped || event.IsInvalid {
			return hash, fmt.Errorf("failed to make call")
		}
	}

	return hash, nil
}

func (s *Substrate) checkForError(blockHash types.Hash, signer types.AccountID) error {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil, nil)
	if err != nil {
		return err
	}

	raw, err := s.cl.RPC.State.GetStorageRaw(key, blockHash)
	if err != nil {
		return err
	}

	block, err := s.cl.RPC.Chain.GetBlock(blockHash)
	if err != nil {
		return err
	}

	events := EventRecords{}
	err = types.EventRecordsRaw(*raw).DecodeEventRecords(meta, &events)
	if err != nil {
		log.Debug().Msgf("Failed to decode event %+v", err)
		return nil
	}

	if len(events.System_ExtrinsicFailed) > 0 {
		for _, e := range events.System_ExtrinsicFailed {
			who := block.Block.Extrinsics[e.Phase.AsApplyExtrinsic].Signature.Signer.AsID
			if signer == who {
				return fmt.Errorf(smartContractModuleErrors[e.DispatchError.Error])
			}
		}
	}

	return nil
}
