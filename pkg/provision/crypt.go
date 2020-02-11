package provision

import (
	"encoding/hex"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func decryptSecret(client zbus.Client, secret string) (string, error) {
	if len(secret) == 0 {
		return "", nil
	}
	identity := stubs.NewIdentityManagerStub(client)

	bytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}

	out, err := identity.Decrypt(bytes)
	return string(out), err
}
