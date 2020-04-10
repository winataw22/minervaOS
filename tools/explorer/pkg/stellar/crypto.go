package stellar

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

type (
	key [32]byte
)

// encryptionKey of the wallet, which is based on the keypair used to create
// this wallet
func (w *Wallet) encryptionKey() key {
	// Annoyingly, we can't get the bytes of the private key, only a string form
	// of the seed. So we might as well hash it again to generate the key.
	return blake2b.Sum256([]byte(w.keypair.Seed()))
}

// encrypt a seed with a given key. The encrypted seed is returned, with the
// generated nonce prepended, encoded in hex form.
//
// The used encryption is AES in GCM mode, with a 32 byte key
func encrypt(seed string, key key) (string, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		// Because we require key to be a byte array of length 32, which is a
		// known valid key, and then slice it, this should never happen
		return "", errors.Wrap(err, "could not setup aes encryption")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "could not set up aes gcm mode")
	}

	// Nonce MUST be unique
	nonce := make([]byte, aesgcm.NonceSize()) // Default nonce size is alway 12
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.Wrap(err, "could not initialize nonce")
	}

	// append the ciphertext to the nonce
	ciphertext := aesgcm.Seal(nonce, nonce, []byte(seed), nil)

	return hex.EncodeToString(ciphertext), err
}

// decrypt a seed which was previously encrypted. The input is expected to be
// the output of the `encrypt` function: a hex encoded string, which starts with
// the nonce, followed by the actual ciphertext.
//
// This function is effectively the inverse of the `encrypt` function
func decrypt(cipherHex string, key key) (string, error) {
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode hex ciphertext")
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		// Because we require key to be a byte array of length 32, which is a
		// known valid key, and then slice it, this should never happen
		return "", errors.Wrap(err, "could not setup aes decryption")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "could not set up aes gcm mode")
	}

	// nonce is prepended to data
	nonceSize := aesgcm.NonceSize()
	plainText, err := aesgcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt seed")
	}

	return string(plainText), nil
}
