package identity

import (
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	keypair, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keypair.PrivateKey)
	assert.NotNil(t, keypair.PublicKey)
}

func TestSerialize(t *testing.T) {
	keypair, err := GenerateKeyPair()
	require.NoError(t, err)

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	err = keypair.Save(f.Name())
	require.NoError(t, err)

	keypair2, err := LoadKeyPair(f.Name())
	require.NoError(t, err)

	assert.Equal(t, keypair.PrivateKey, keypair2.PrivateKey)
	assert.Equal(t, keypair.PublicKey, keypair2.PublicKey)
}

func TestIdentity(t *testing.T) {
	sk := ed25519.NewKeyFromSeed([]byte("helloworldhelloworldhelloworld12"))
	keypair := KeyPair{
		PrivateKey: sk,
		PublicKey:  sk.Public().(ed25519.PublicKey),
	}
	id := keypair.Identity()
	assert.Equal(t, "FkUfMueBVSK6V1DCHVAtzzaqPqCPVzGguDzCQxq7Ep85", id)
	assert.Equal(t, "FkUfMueBVSK6V1DCHVAtzzaqPqCPVzGguDzCQxq7Ep85", url.PathEscape(id), "identity should be url friendly")
}
