package identity

import (
	"io/ioutil"
	"os"
	"testing"

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

	err = SerializeSeed(keypair, f.Name())
	require.NoError(t, err)

	keypair2, err := LoadSeed(f.Name())
	require.NoError(t, err)

	assert.Equal(t, keypair.PrivateKey, keypair2.PrivateKey)
	assert.Equal(t, keypair.PublicKey, keypair2.PublicKey)
}
