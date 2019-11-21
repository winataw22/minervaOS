package provision

import (
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

// TfgridReservationNetwork1 jsx Schema
type TfgridReservationNetwork1 struct {
	Name             string                              `json:"name"`
	WorkloadID       int64                               `json:"workload_id"`
	Iprange          schema.IPRange                      `json:"iprange"`
	StatsAggregator  []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	NetworkResources []TfgridNetworkNetResource1         `json:"network_resources"`
}

// ToProvisionType convert TfgridReservationNetwork1 to pkg.Network
func (n TfgridReservationNetwork1) ToProvisionType() (pkg.Network, error) {
	network := pkg.Network{
		Name:         n.Name,
		NetID:        pkg.NetID(n.Name),
		IPRange:      types.NewIPNetFromSchema(n.Iprange),
		NetResources: make([]pkg.NetResource, len(n.NetworkResources)),
	}

	var err error
	for i, nr := range n.NetworkResources {
		network.NetResources[i], err = nr.ToProvisionType()
		if err != nil {
			return network, err
		}
	}
	return network, nil
}

// TfgridNetworkNetResource1 jsx Schema
type TfgridNetworkNetResource1 struct {
	NodeID                       string           `json:"node_id"`
	IPRange                      schema.IPRange   `json:"iprange"`
	WireguardPrivateKeyEncrypted string           `json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string           `json:"wireguard_public_key"`
	WireguardListenPort          int64            `json:"wireguard_listen_port"`
	Peers                        []WireguardPeer1 `json:"peers"`
}

//ToProvisionType converts TfgridNetworkNetResource1 to pkg.NetResource
func (r TfgridNetworkNetResource1) ToProvisionType() (pkg.NetResource, error) {
	nr := pkg.NetResource{
		NodeID:       r.NodeID,
		Subnet:       types.NewIPNetFromSchema(r.IPRange),
		WGPrivateKey: r.WireguardPrivateKeyEncrypted,
		WGPublicKey:  r.WireguardPublicKey,
		WGListenPort: uint16(r.WireguardListenPort),
		Peers:        make([]pkg.Peer, len(r.Peers)),
	}

	for i, peer := range r.Peers {
		p, err := peer.ToProvisionType()
		if err != nil {
			return nr, err
		}
		nr.Peers[i] = p
	}

	return nr, nil
}

// WireguardPeer1 jsx Schema
type WireguardPeer1 struct {
	PublicKey  string         `json:"public_key"`
	Endpoint   string         `json:"endpoint"`
	AllowedIPs []string       `json:"allowed_iprange"`
	IPRange    schema.IPRange `json:"iprange"`
}

//ToProvisionType converts WireguardPeer1 to pkg.Peer
func (p WireguardPeer1) ToProvisionType() (pkg.Peer, error) {
	peer := pkg.Peer{
		WGPublicKey: p.PublicKey,
		Endpoint:    p.Endpoint,
		AllowedIPs:  make([]types.IPNet, len(p.AllowedIPs)),
		Subnet:      types.NewIPNetFromSchema(p.IPRange),
	}

	var err error
	for i, ip := range p.AllowedIPs {
		peer.AllowedIPs[i], err = types.ParseIPNet(ip)
		if err != nil {
			return peer, err
		}
	}
	return peer, nil
}
