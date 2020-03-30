package stellar

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	usedPlaintext = "very random seed by default"
)

func TestEncrypt(t *testing.T) {
	var key key

	_, err := io.ReadFull(rand.Reader, key[:])
	assert.NoError(t, err, "initializing random key")

	ciphertext, err := encrypt(usedPlaintext, key)
	assert.NoError(t, err, "encrypting random seed")

	// 12 byte nonce, len(seed) byte seed, 16 byte tag, hex encoded => double size
	expectedCiphertextLen := (12 + len(usedPlaintext) + 16) * 2
	assert.Equal(t, expectedCiphertextLen, len(ciphertext), "expected and actual ciphertext len")
}

func TestDecrypt(t *testing.T) {
	keyHex := "69340634a02dc317f4777246718a589f0879969eb9c0c730858e5c8e5e4fcbff"
	ciphertext := "0de81f80d88426a431f2bb92699e470f14154d9283156f73a6024c80991e2bca57bbfab08c0b24052e5d1fb242c04253d6ddd73793b9f6"

	keyBytes, err := hex.DecodeString(keyHex)
	assert.NoError(t, err, "could not decode key")

	assert.Equal(t, 32, len(keyBytes))

	var key key
	copy(key[:], keyBytes)

	plaintext, err := decrypt(ciphertext, key)
	assert.NoError(t, err, "could not decrypt ciphertext")

	assert.Equal(t, usedPlaintext, plaintext, "expected and actual plaintext don't match")
}

func TestEncryptionRoundTrip(t *testing.T) {
	plaintext := "Very secret stuff goes here"

	var key key
	_, err := io.ReadFull(rand.Reader, key[:])
	assert.NoError(t, err, "could not generate key")

	ciphertext, err := encrypt(plaintext, key)
	assert.NoError(t, err, "could not encrypt plainText")

	reversedPlaintext, err := decrypt(ciphertext, key)
	assert.NoError(t, err, "could not decrypt cipherText")

	assert.Equal(t, plaintext, reversedPlaintext, "original and decrypted plaintext mismatch")

}
