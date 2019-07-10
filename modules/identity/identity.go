package identity

// Identifier is the interface that defines
// how an object can be used an identity
type Identifier interface {
	Identity() string
}

// strIdentifier is a helper type that implement the Identifier interface
// on top of simple string
type strIdentifier string

// Identity implements the Identifier interface
func (s strIdentifier) Identity() string {
	return string(s)
}
