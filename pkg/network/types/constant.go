package types

const (
	// PublicNamespace is the name of the public namespace of a node
	// the public namespace is currently unique for a node so we hardcode its name
	PublicNamespace = "public"
	// PublicIface is the name of the interface we create in the public namespace
	PublicIface = "public"
	// GatewayNamespace is the name of the gateway namespace of a node
	GatewayNamespace = "gateway"
	// DefaultBridge is the name of the default bridge created
	// by the bootstrap of networkd
	DefaultBridge = "zos"
	// PublicBridge name
	PublicBridge = "br-pub"
	// YggBridge ygg bridge
	YggBridge = "br-ygg"
	// MyceliumBridge mycelium bridge
	MyceliumBridge = "br-my"
)
