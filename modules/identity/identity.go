package identity

import "encoding/base64"

// Identifier is the interface that defines
// how an object can be used an an identity for
// node and farm
type Identifier interface {
	Identity() string
}

// NodeID represent the identity of a node
type NodeID struct {
	keyPair *KeyPair
}

// NewNodeID creates a new node identity based on a key pair
func NewNodeID(keyPair *KeyPair) *NodeID {
	return &NodeID{keyPair}
}

// Identity implements the Identifier interface
func (n *NodeID) Identity() string {
	return base64.StdEncoding.EncodeToString(n.keyPair.PublicKey)
}

// Farm is the struct holding the detail about a Farm
type Farm struct {
	name    string
	keyPair *KeyPair
}

// NewFarm creates a new farm object
func NewFarm(name string, keypair *KeyPair) *Farm {
	return &Farm{
		name:    name,
		keyPair: keypair,
	}
}

// Identity implements the Identifier interface
func (f *Farm) Identity() string {
	return base64.StdEncoding.EncodeToString(f.keyPair.PublicKey)
}

// Name returns the name of the farm
func (f *Farm) Name() string {
	return f.name
}

// strIdentifier is a helper type that implement the Identifier interface
// on top of simple string
type strIdentifier string

// Identity implements the Identifier interface
func (s strIdentifier) Identity() string {
	return string(s)
}
