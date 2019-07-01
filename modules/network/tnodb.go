package network

import (
	"net"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/identity"
)

// TNoDB define the interface to implement
// to talk to a Tenant Network object database
type TNoDB interface {
	RegisterAllocation(farm identity.Identifier, allocation *net.IPNet) error
	RequestAllocation(farm identity.Identifier) (*net.IPNet, error)
	PublishInterfaces() error
	ConfigureExitNode(node identity.Identifier, ip *net.IPNet, gw net.IP, iface string) error
	ReadExitNode(node identity.Identifier) (*ExitIface, error)
	// ReadNetworkObj(node identity.Identifier) ([]*modules.Network, error)
	PublishWireguarKey(string, modules.NodeID, modules.NetID) error
}
