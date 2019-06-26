package identity

import "encoding/base64"

// Identifier is the interface that defines
// how an object can be used an an identity for
// node and farm
type Identifier interface {
	Identity() string
}

type NodeID struct {
	keyPair *KeyPair
}

func NewNodeID(keyPair *KeyPair) *NodeID {
	return &NodeID{keyPair}
}

func (n *NodeID) Identity() string {
	return base64.StdEncoding.EncodeToString(n.keyPair.PublicKey)
}

type Farm struct {
	name    string
	keyPair *KeyPair
}

func NewFarm(name string, keypair *KeyPair) *Farm {
	return &Farm{
		name:    name,
		keyPair: keypair,
	}
}

func (n *Farm) Identity() string {
	return base64.StdEncoding.EncodeToString(n.keyPair.PublicKey)
}

func (f *Farm) Name() string {
	return f.name
}
