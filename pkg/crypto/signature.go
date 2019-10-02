package crypto

import (
	"fmt"

	"golang.org/x/crypto/ed25519"
)

// Verify reports whether sig is a valid signature of message by publicKey.
func Verify(publicKey ed25519.PublicKey, message, sig []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("public key has the wrong size")
	}
	if !ed25519.Verify(publicKey, message, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// Sign signs the message with privateKey and returns a signature.
func Sign(privateKey ed25519.PrivateKey, message []byte) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key has the wrong size")
	}
	return ed25519.Sign(privateKey, message), nil
}
