package substrate

import (
	"crypto/ed25519"

	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
)

// AccountID type
type AccountID types.AccountID

//PublicKey gets public key from account id
func (a AccountID) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(a[:])
}
