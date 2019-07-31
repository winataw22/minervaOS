package crypto

import (
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zosv2/modules"
	"golang.org/x/crypto/ed25519"
)

// KeyFromID extract the public key from an Identifier
func KeyFromID(id modules.Identifier) (ed25519.PublicKey, error) {
	b := base58.Decode(id.Identity())
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wrong key size")
	}
	return ed25519.PublicKey(b), nil
}
