package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/schema"
)

// Signer is a utility to easily sign payloads
type Signer struct {
	pair identity.KeyPair
}

// NewSigner create a signer with a seed
func NewSigner(seed []byte) (*Signer, error) {
	pair, err := identity.FromSeed(seed)
	if err != nil {
		return nil, err
	}

	return &Signer{pair: pair}, nil
}

// NewSignerFromFile loads signer from a seed file
func NewSignerFromFile(path string) (*Signer, error) {
	pair, err := identity.LoadKeyPair(path)

	if err != nil {
		return nil, err
	}

	return &Signer{pair: pair}, nil
}

// SignHex like sign, but return message and signature in hex encoded format
func (s *Signer) SignHex(o ...interface{}) (string, string, error) {
	msg, sig, err := s.Sign(o...)
	if err != nil {
		return "", "", err
	}

	return hex.EncodeToString(msg), hex.EncodeToString(sig), nil
}

// Sign constructs a message from all it's arguments then sign it
func (s *Signer) Sign(o ...interface{}) ([]byte, []byte, error) {
	var buf bytes.Buffer
	for _, x := range o {
		switch x := x.(type) {
		case nil:
		case string:
			buf.WriteString(x)
		case fmt.Stringer:
			buf.WriteString(x.String())
		case []byte:
			buf.Write(x)
		case json.RawMessage:
			buf.Write(x)
		case byte:
			buf.WriteString(fmt.Sprint(x))
		// all int types
		case schema.ID:
			buf.WriteString(fmt.Sprint(x))
		case int:
			buf.WriteString(fmt.Sprint(x))
		case int8:
			buf.WriteString(fmt.Sprint(x))
		case int16:
			buf.WriteString(fmt.Sprint(x))
		case int32:
			buf.WriteString(fmt.Sprint(x))
		case int64:
			buf.WriteString(fmt.Sprint(x))
		// all float types
		case float32:
			buf.WriteString(fmt.Sprint(x))
		case float64:
			buf.WriteString(fmt.Sprint(x))
		default:
			return nil, nil, fmt.Errorf("unsupported type")
		}
	}
	msg := buf.Bytes()
	sig, err := crypto.Sign(s.pair.PrivateKey, msg)
	return msg, sig, err
}
