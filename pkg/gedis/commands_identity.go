package gedis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jbenet/go-base58"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/geoip"
)

//
// IDStore Interface
//

//RegisterNode implements pkg.IdentityManager interface
func (g *Gedis) RegisterNode(nodeID pkg.Identifier, farmID pkg.FarmID, version string, location geoip.Location) (string, error) {

	pk := base58.Decode(nodeID.Identity())
	node := directory.TfgridNode2{
		NodeID:       nodeID.Identity(),
		FarmID:       uint64(farmID),
		OsVersion:    version,
		PublicKeyHex: hex.EncodeToString(pk),
		Location: directory.TfgridLocation1{
			Longitude: location.Longitute,
			Latitude:  location.Latitude,
			Continent: location.Continent,
			Country:   location.Country,
			City:      location.City,
		},
	}
	idv1, err := network.NodeIDv1()
	if err == nil {
		node.NodeIDv1 = idv1
	}

	resp, err := Bytes(g.Send("tfgrid.directory.nodes", "add", Args{"node": node}))

	if err != nil {
		return "", err
	}

	var out directory.TfgridNode2

	if err := json.Unmarshal(resp, &out); err != nil {
		return "", err
	}

	// no need to do data conversion here, returns the id
	return out.NodeID, nil
}

// ListNode implements pkg.IdentityManager interface
func (g *Gedis) ListNode(farmID pkg.FarmID, country string, city string) ([]types.Node, error) {
	resp, err := Bytes(g.Send("tfgrid.directory.nodes", "list", Args{
		"farm_id": farmID,
		"country": country,
		"city":    city,
	}))

	if err != nil {
		return nil, err
	}

	var out struct {
		Nodes []directory.TfgridNode2 `json:"nodes"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	var result []types.Node
	for _, node := range out.Nodes {
		result = append(result, nodeFromSchema(node))
	}

	return result, nil
}

//RegisterFarm implements pkg.IdentityManager interface
func (g *Gedis) RegisterFarm(tid uint64, name string, email string, wallet []string) (pkg.FarmID, error) {
	resp, err := Bytes(g.Send("tfgrid.directory.farms", "register", Args{
		"farm": directory.TfgridFarm1{
			ThreebotID:      tid,
			Name:            name,
			Email:           email,
			WalletAddresses: wallet,
		},
	}))

	if err != nil {
		return 0, err
	}

	var out struct {
		FarmID json.Number `json:"farm_id"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return 0, err
	}

	id, err := out.FarmID.Int64()
	if err != nil {
		return 0, err
	}

	return pkg.FarmID(id), nil
}

//GetNode implements pkg.IdentityManager interface
func (g *Gedis) GetNode(nodeID pkg.Identifier) (*types.Node, error) {
	resp, err := Bytes(g.Send("tfgrid.directory.nodes", "get", Args{
		"node_id": nodeID.Identity(),
	}))

	if err != nil {
		return nil, err
	}

	var out directory.TfgridNode2

	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	node := nodeFromSchema(out)
	return &node, nil
}

func infFromSchema(inf directory.TfgridNodeIface1) types.IfaceInfo {
	return types.IfaceInfo{
		Name:    inf.Name,
		Gateway: inf.Gateway,
		Addrs: func() []types.IPNet {
			var r []types.IPNet
			for _, addr := range inf.Addrs {
				r = append(r, types.NewIPNetFromSchema(addr))
			}
			return r
		}(),
	}
}

func nodeFromSchema(node directory.TfgridNode2) types.Node {
	return types.Node{
		NodeID: node.NodeID,
		FarmID: node.FarmID,
		Ifaces: func() []*types.IfaceInfo {
			var r []*types.IfaceInfo
			for _, iface := range node.Ifaces {
				v := infFromSchema(iface)
				r = append(r, &v)
			}
			return r
		}(),
		PublicConfig: func() *types.PubIface {
			cfg := node.PublicConfig
			if cfg == nil || cfg.Master == "" {
				return nil
			}
			pub := types.PubIface{
				Master:  cfg.Master,
				Type:    types.IfaceType(cfg.Type.String()),
				IPv4:    types.NewIPNetFromSchema(cfg.Ipv4),
				IPv6:    types.NewIPNetFromSchema(cfg.Ipv6),
				GW4:     cfg.Gw4,
				GW6:     cfg.Gw6,
				Version: int(cfg.Version),
			}

			return &pub
		}(),
		WGPorts: node.WGPorts,
	}
}

func farmFromSchema(farm directory.TfgridFarm1) network.Farm {
	return network.Farm{
		ID:   fmt.Sprint(farm.ID),
		Name: farm.Name,
	}
}

//GetFarm implements pkg.IdentityManager interface
func (g *Gedis) GetFarm(farm pkg.Identifier) (*network.Farm, error) {
	id, err := strconv.ParseInt(farm.Identity(), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid farm id")
	}

	resp, err := Bytes(g.Send("tfgrid.directory.farms", "get", Args{
		"farm_id": id,
	}))

	if err != nil {
		return nil, err
	}

	var out directory.TfgridFarm1

	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}
	f := farmFromSchema(out)
	return &f, nil
}

//ListFarm implements pkg.IdentityManager interface
func (g *Gedis) ListFarm(country string, city string) ([]network.Farm, error) {
	resp, err := Bytes(g.Send("tfgrid.directory.farms", "list", Args{
		"country": country,
		"city":    city,
	}))

	if err != nil {
		return nil, err
	}

	var out struct {
		Farms []directory.TfgridFarm1 `json:"farms"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	var result []network.Farm
	for _, farm := range out.Farms {
		result = append(result, farmFromSchema(farm))
	}

	return result, nil
}
