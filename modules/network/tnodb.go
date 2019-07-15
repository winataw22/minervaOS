package network

import (
	"errors"
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

	ConfigurePublicIface(node identity.Identifier, ip *net.IPNet, gw net.IP, iface string) error
	ReadPubIface(node identity.Identifier) (*PubIface, error)

	SelectExitNode(node identity.Identifier) error

	CreateNetwork(farmID string) (*modules.Network, error)
	GetNetwork(netID modules.NetID) (*modules.Network, error)

	PublishWireguarKey(key string, nodeID string, netID modules.NetID) error
}

// ErrNoPubIface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubIface = errors.New("no public interface configured for this node")
