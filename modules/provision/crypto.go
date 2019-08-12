package provision

import (
	"bytes"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/crypto"
	"golang.org/x/crypto/ed25519"
)

// Sign creates a signature from all the field of the reservation
// object and fill the Signature field
func (r *Reservation) Sign(privateKey ed25519.PrivateKey) error {
	buf := &bytes.Buffer{}
	//FIME: Since the ID is only set when the reservation is sent to bcdb
	// we cannot use it in the signature. This is a problem

	// _, err := buf.WriteString(r.ID)
	// if err != nil {
	// 	return err
	// }
	_, err := buf.WriteString(r.User)
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
func Verify(r *Reservation) error {
	buf := &bytes.Buffer{}
	//FIME: Since the ID is only set when the reservation is sent to bcdb
	// we cannot use it in the signature. This is a problem

	_, err := buf.WriteString(r.User)
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

	publicKey, err := crypto.KeyFromID(modules.StrIdentifier(r.User))
	if err != nil {
		return errors.Wrap(err, "failed to extract public key from user ID")
	}

	return crypto.Verify(publicKey, buf.Bytes(), r.Signature)
}
