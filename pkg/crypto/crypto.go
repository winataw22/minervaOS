package crypto

import (
	"encoding/hex"
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"golang.org/x/crypto/ed25519"
)

// KeyFromID extract the public key from an Identifier
func KeyFromID(id pkg.Identifier) (ed25519.PublicKey, error) {
	b := base58.Decode(id.Identity())
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wrong key size")
	}
	return ed25519.PublicKey(b), nil
}

// KeyFromHex extract the public key from a hex string (used with jsx keys)
func KeyFromHex(h string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, errors.Wrap(err, "invalid hex format")
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wrong key size")
	}
	return ed25519.PublicKey(b), nil
}
