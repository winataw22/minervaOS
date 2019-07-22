package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
)

func TestEncrypt(t *testing.T) {
	alicePk, aliceSk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	msg := []byte("hello world")

	cipher, err := Encrypt(msg, alicePk)
	require.NoError(t, err)
	clear, err := Decrypt(cipher, aliceSk)
	assert.Equal(t, clear, msg)
}
