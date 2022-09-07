package identity

import (
	"net/url"
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
