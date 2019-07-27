package provision

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/identity"

	"github.com/threefoldtech/zosv2/modules/crypto"
	"golang.org/x/crypto/ed25519"
)

// Sign creates a signature from all the field of the reservation
// object and fill the Signature field
func (r *Reservation) Sign(privateKey ed25519.PrivateKey) error {
	buf := &bytes.Buffer{}
	_, err := buf.WriteString(r.ID)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(r.User)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(string(r.Type))
	if err != nil {
		return err
	}
	_, err = buf.Write(r.Data)
	if err != nil {
		return err
	}

	signature, err := crypto.Sign(privateKey, buf.Bytes())
	if err != nil {
		return err
	}
	r.Signature = signature
	return nil
}

// Verify verifies the signature of the reservation
func Verify(r Reservation) error {
	buf := &bytes.Buffer{}
	_, err := buf.WriteString(r.ID)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(r.User)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(string(r.Type))
	if err != nil {
		return err
	}
	_, err = buf.Write(r.Data)
	if err != nil {
		return err
	}

	publicKey, err := crypto.KeyFromID(identity.StrIdentifier(r.User))
	if err != nil {
		return errors.Wrap(err, "failed to extract public key from user ID")
	}

	return crypto.Verify(publicKey, buf.Bytes(), r.Signature)
}

func Hash(r Reservation) ([]byte, error) {
	h := sha256.New()
	_, err := io.WriteString(h, r.ID)
	if err != nil {
		return nil, err
	}
	_, err = io.WriteString(h, r.User)
	if err != nil {
		return nil, err
	}
	_, err = io.WriteString(h, string(r.Type))
	if err != nil {
		return nil, err
	}
	_, err = h.Write(r.Data)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func HexHash(r Reservation) (string, error) {
	h, err := Hash(r)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h), nil
}
