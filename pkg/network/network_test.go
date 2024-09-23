package network

import (
	"crypto/rand"
	"fmt"
	"testing"

	"golang.org/x/crypto/ed25519"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/threefoldtech/zos/pkg/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	wgKey, err := wgtypes.GenerateKey()
	require.NoError(t, err)

	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	fmt.Println(wgKey.String())

	encrypted, err := crypto.Encrypt([]byte(wgKey.String()), pk)
	require.NoError(t, err)

	strEncrypted := fmt.Sprintf("%x", encrypted)

	strDecrypted := ""
	_, err = fmt.Sscanf(strEncrypted, "%x", &strDecrypted)
	require.NoError(t, err)

	decrypted, err := crypto.Decrypt([]byte(strDecrypted), sk)
	require.NoError(t, err)

	fmt.Println(string(decrypted))

	wgKey2, err := wgtypes.ParseKey(string(decrypted))
	require.NoError(t, err)

	assert.Equal(t, wgKey, wgKey2)
}
