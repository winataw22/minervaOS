package primitives

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func decryptSecret(secret, userID string, reservationVersion int, client zbus.Client) (string, error) {
	if len(secret) == 0 {
		return "", nil
	}

	identity := stubs.NewIdentityManagerStub(client)

	bytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}

	var (
		out        []byte
		userPubKey ed25519.PublicKey
	)
	switch reservationVersion {
	case 0:
		out, err = identity.Decrypt(bytes)
	case 1:
		userPubKey, err = fetchUserPublicKey(userID)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve user %s public key: %w", userID, err)
		}
		out, err = identity.DecryptECDH(bytes, userPubKey)
	}

	return string(out), err
}

func fetchUserPublicKey(userID string) (ed25519.PublicKey, error) {
	iid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, err
	}

	explorer, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	user, err := explorer.Phonebook.Get(schema.ID(iid))
	if err != nil {
		return nil, err
	}

	b, err := hex.DecodeString(user.Pubkey)
	if err != nil {
		return nil, err
	}

	return ed25519.PublicKey(b), nil
}
