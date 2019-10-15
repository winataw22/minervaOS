package gedis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jbenet/go-base58"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/geoip"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg"
)

//
// IDStore Interface
//

//RegisterNode implements pkg.IdentityManager interface
func (g *Gedis) RegisterNode(nodeID, farmID pkg.Identifier, version string) (string, error) {

	l, err := geoip.Fetch()
	if err != nil {
		log.Error().Err(err).Msg("failed to get location of the node")
	}

	pk := base58.Decode(nodeID.Identity())

	resp, err := Bytes(g.Send("nodes", "add", Args{
		"node": directory.TfgridNode2{
			NodeID:       nodeID.Identity(),
			FarmID:       farmID.Identity(),
			OsVersion:    version,
			PublicKeyHex: hex.EncodeToString(pk),
			Location: directory.TfgridLocation1{
				Longitude: l.Longitute,
				Latitude:  l.Latitude,
				Continent: l.Continent,
				Country:   l.Country,
				City:      l.City,
			},
		},
	}))

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
func (g *Gedis) ListNode(farmID pkg.Identifier, country string, city string) ([]types.Node, error) {
	resp, err := Bytes(g.Send("nodes", "list", Args{
		"farm_id": farmID.Identity(),
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
func (g *Gedis) RegisterFarm(farm pkg.Identifier, name string, email string, wallet []string) (string, error) {
	resp, err := Bytes(g.Send("farms", "register", Args{
		"farm": directory.TfgridFarm1{
			ThreebotID:      farm.Identity(),
			Name:            name,
			Email:           email,
			WalletAddresses: wallet,
		},
	}))

	if err != nil {
		return "", err
	}

	var out struct {
		FarmID json.Number `json:"farm_id"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return "", err
	}

	return out.FarmID.String(), nil
}

//GetNode implements pkg.IdentityManager interface
func (g *Gedis) GetNode(nodeID pkg.Identifier) (*types.Node, error) {
	resp, err := Bytes(g.Send("nodes", "get", Args{
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
			// This is a dirty hack because jsx schema cannot
			// differentiate between an embed object not set or with default value
			if cfg.Master == "" {
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
		ExitNode: func() int {
			if node.ExitNode {
				return 1
			}
			return 0
		}(),
	}
}

func farmFromSchema(farm directory.TfgridFarm1) network.Farm {
	return network.Farm{
		ID:   fmt.Sprint(farm.ID),
		Name: farm.Name,
	}
}

func (g *Gedis) updateGenericNodeCapacity(captype string, node pkg.Identifier, mru, cru, hru, sru uint64) error {
	_, err := g.Send("nodes", "update_"+captype+"_capacity", Args{
		"node_id": node.Identity(),
		"resource": directory.TfgridNodeResourceAmount1{
			Cru: int64(cru),
			Mru: int64(mru),
			Hru: int64(hru),
			Sru: int64(sru),
		},
	})

	return err
}

//UpdateTotalNodeCapacity implements pkg.IdentityManager interface
func (g *Gedis) UpdateTotalNodeCapacity(node pkg.Identifier, mru, cru, hru, sru uint64) error {
	return g.updateGenericNodeCapacity("total", node, mru, cru, hru, sru)
}

//UpdateReservedNodeCapacity implements pkg.IdentityManager interface
func (g *Gedis) UpdateReservedNodeCapacity(node pkg.Identifier, mru, cru, hru, sru uint64) error {
	return g.updateGenericNodeCapacity("reserved", node, mru, cru, hru, sru)
}

//UpdateUsedNodeCapacity implements pkg.IdentityManager interface
func (g *Gedis) UpdateUsedNodeCapacity(node pkg.Identifier, mru, cru, hru, sru uint64) error {
	return g.updateGenericNodeCapacity("used", node, mru, cru, hru, sru)
}

//GetFarm implements pkg.IdentityManager interface
func (g *Gedis) GetFarm(farm pkg.Identifier) (*network.Farm, error) {
	id, err := strconv.ParseInt(farm.Identity(), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid farm id")
	}

	resp, err := Bytes(g.Send("farms", "get", Args{
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
	resp, err := Bytes(g.Send("farms", "list", Args{
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
