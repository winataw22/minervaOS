package gridtypes

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

type polkaSigner struct {
	Signer
}

func (s *polkaSigner) Sign(msg []byte) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.WriteString("<Bytes>"); err != nil {
		return nil, err
	}

	if _, err := buf.WriteString(hex.EncodeToString(msg)); err != nil {
		return nil, err
	}

	if _, err := buf.WriteString("</Bytes>"); err != nil {
		return nil, err
	}

	return s.Signer.Sign(buf.Bytes())
}

type keyGetter struct {
	twin uint32
	key  []byte
}

func (k *keyGetter) GetKey(twin uint32) ([]byte, error) {
	if k.twin != twin {
		return nil, fmt.Errorf("unknown twin")
	}

	return k.key, nil
}

func TestSignVerify(t *testing.T) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	id, err := substrate.NewIdentityFromEd25519Key(sk)
	require.NoError(t, err)

	signer := polkaSigner{id}

	dl := Deployment{
		TwinID: 1,
		SignatureRequirement: SignatureRequirement{
			Requests: []SignatureRequest{
				{TwinID: 1, Required: true, Weight: 1},
			},
			SignatureStyle: SignatureStylePolka,
		},
	}

	require.NoError(t, dl.Sign(1, &signer))

	getter := keyGetter{twin: 1, key: pk}
	require.NoError(t, dl.Verify(&getter))
}
