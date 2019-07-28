package modules

// Identifier is the interface that defines
// how an object can be used an identity
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
	FarmID() (StrIdentifier, error)

	// Sign
	Sign(data []byte) ([]byte, error)

	// Verify
	Verify(data, sig []byte) error

	// Encrypt, Decrypt ?
}
