package substrate

import (
	"crypto/ed25519"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/vedhavyas/go-subkey"
	subkeyEd25519 "github.com/vedhavyas/go-subkey/ed25519"
	"golang.org/x/crypto/blake2b"
)

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
func (s *Substrate) sign(e *types.Extrinsic, signer signature.KeyringPair, o types.SignatureOptions) error {
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

func (s *Substrate) call(sk ed25519.PrivateKey, call types.Call) error {

	// Create the extrinsic
	ext := types.NewExtrinsic(call)

	genesisHash, err := s.cl.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "failed to get genesisHash")
	}

	rv, err := s.cl.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return err
	}

	identity, err := Identity(sk)
	if err != nil {
		return err
	}

	//node.Address =identity.PublicKey
	account, err := s.getAccount(identity, s.meta)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
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
		return errors.Wrap(err, "failed to sign")
	}

	// Send the extrinsic
	sub, err := s.cl.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "failed to submit extrinsic")
	}

	defer sub.Unsubscribe()

	for event := range sub.Chan() {
		if event.IsFinalized {
			break
		} else if event.IsDropped || event.IsInvalid {
			return fmt.Errorf("failed to make call")
		}
	}

	return nil
}
