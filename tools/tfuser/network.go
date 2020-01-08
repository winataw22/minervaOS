package main

import (
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// Network represent the description if a user private network
type Network struct {
	pkg.Network

	AccessPoints []AccessPoint `json:"access_points,omitempty"`

	// NetResources field override
	NetResources []NetResource `json:"net_resources"`
}

// NetResource is the description of a part of a network local to a specific node
type NetResource struct {
	pkg.NetResource

	// Public endpoints
	PubEndpoints []net.IP `json:"pub_endpoints"`
}

// AccessPoint info for a network, defining a node which will act as the AP, and
// the subnet which will be routed through it
type AccessPoint struct {
	// NodeID of the access point in the network
	NodeID string `json:"node_id"`
	// Subnet to be routed through this access point
	Subnet      types.IPNet `json:"subnet"`
	WGPublicKey string      `json:"wg_public_key"`
	IP4         bool        `json:"ip4"`
}

func networkFromPkgNet(network pkg.Network) Network {
	n := Network{Network: network}

	nrs := make([]NetResource, 0, len(network.NetResources))
	for _, nr := range network.NetResources {
		nrs = append(nrs, NetResource{NetResource: nr})
	}
	n.NetResources = nrs

	return n
}

func pkgNetFromNetwork(network Network) pkg.Network {
	n := pkg.Network{
		Name:    network.Name,
		NetID:   network.NetID,
		IPRange: network.IPRange,
	}

	nrs := make([]pkg.NetResource, 0, len(network.NetResources))
	for _, nr := range network.NetResources {
		nrs = append(nrs, pkg.NetResource{
			NodeID:       nr.NodeID,
			WGPrivateKey: nr.WGPrivateKey,
			WGPublicKey:  nr.WGPublicKey,
			WGListenPort: nr.WGListenPort,
			Subnet:       nr.Subnet,
			Peers:        nr.Peers,
		})
	}
	n.NetResources = nrs

	return n
}
