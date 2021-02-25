package primitives

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (p *Primitives) decryptSecret(ctx context.Context, user gridtypes.ID, secret string, version int) (string, error) {
	if len(secret) == 0 {
		return "", nil
	}

	engine := provision.GetEngine(ctx)

	identity := stubs.NewIdentityManagerStub(p.zbus)

	bytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}

	var (
		out []byte
	)
	// now only one version is supported
	switch version {
	default:
		var userPubKey ed25519.PublicKey
		userPubKey, err = engine.Users().GetKey(user)
		if err != nil || userPubKey == nil {
			return "", fmt.Errorf("failed to retrieve user %s public key: %s", user, err)
		}
		out, err = identity.DecryptECDH(bytes, userPubKey)
	}

	return string(out), err
}
