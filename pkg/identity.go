package pkg

//go:generate mkdir -p stubs
//go:generate zbusc -module identityd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+IdentityManager stubs/identity_stub.go

// Identifier is the interface that defines
// how an object can be used as an identity
type Identifier interface {
	Identity() string
}

// StrIdentifier is a helper type that implement the Identifier interface
// on top of simple string
type StrIdentifier string

// Identity implements the Identifier interface
func (s StrIdentifier) Identity() string {
	return string(s)
}

// IdentityManager interface.
type IdentityManager interface {
	// NodeID returns the node id (public key)
	NodeID() StrIdentifier

	// FarmID return the farm id this node is part of. this is usually a configuration
	// that the node is booted with. An error is returned if the farmer id is not configured
	FarmID() (FarmID, error)

	// Sign signs the message with privateKey and returns a signature.
	Sign(message []byte) ([]byte, error)

	// Verify reports whether sig is a valid signature of message by publicKey.
	Verify(message, sig []byte) error

	// Encrypt encrypts message with the public key of the node
	Encrypt(message []byte) ([]byte, error)

	// Decrypt decrypts message with the private of the node
	Decrypt(message []byte) ([]byte, error)

	// PrivateKey sends the keypair
	PrivateKey() []byte
}

// FarmID is the identification of a farm
type FarmID uint64
