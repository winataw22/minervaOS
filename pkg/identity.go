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

type Address string

func (s Address) String() string {
	return string(s)
}

// IdentityManager interface.
type IdentityManager interface {
	// Store returns the key store kind
	StoreKind() string

	// NodeID returns the node id (public key)
	NodeID() StrIdentifier

	// Address return the node address (SS58Address address)
	Address() (Address, error)

	// FarmID return the farm id this node is part of. this is usually a configuration
	// that the node is booted with. An error is returned if the farmer id is not configured
	FarmID() (FarmID, error)

	// Farm returns name of the farm. Or error
	Farm() (string, error)

	//FarmSecret get the farm secret as defined in the boot params
	FarmSecret() (string, error)

	// Sign signs the message with privateKey and returns a signature.
	Sign(message []byte) ([]byte, error)

	// Verify reports whether sig is a valid signature of message by publicKey.
	Verify(message, sig []byte) error

	// Encrypt encrypts message with the public key of the node
	Encrypt(message []byte) ([]byte, error)

	// Decrypt decrypts message with the private of the node
	Decrypt(message []byte) ([]byte, error)

	// EncryptECDH aes encrypt msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
	// the nonce if prepended to the encrypted message
	EncryptECDH(msg []byte, publicKey []byte) ([]byte, error)

	// DecryptECDH decrypt aes encrypted msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
	DecryptECDH(msg []byte, publicKey []byte) ([]byte, error)

	// PrivateKey sends the keypair
	PrivateKey() []byte
}

// FarmID is the identification of a farm
type FarmID uint32
