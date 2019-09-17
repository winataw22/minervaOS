package gedis

import (
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/types"

	"github.com/garyburd/redigo/redis"
	"github.com/threefoldtech/zosv2/modules"
)

//
// IDStore Interface
//

func (g *Gedis) sendCommand(actor string, method string, b []byte) ([]byte, error) {
	con := g.pool.Get()
	defer con.Close()

	resp, err := redis.Bytes(con.Do(g.cmd(actor, method), b, g.headers))
	if err != nil {
		return []byte{}, err
	}

	return resp, nil
}

func (g *Gedis) sendCommandBool(actor string, method string, b []byte) (bool, error) {
	con := g.pool.Get()
	defer con.Close()

	resp, err := redis.Bool(con.Do(g.cmd(actor, method), b, g.headers))
	if err != nil {
		return false, err
	}

	return resp, nil
}

//RegisterNode implements modules.IdentityManager interface
func (g *Gedis) RegisterNode(nodeID, farmID modules.Identifier, version string) (string, error) {
	req := gedisRegisterNodeBody{
		Node: tfgridNode2{
			NodeID:  nodeID.Identity(),
			FarmID:  farmID.Identity(),
			Version: version,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := g.sendCommand("nodes", "add", b)
	if err != nil {
		return "", parseError(err)
	}

	r := tfgridNode2{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}

	// no need to do data conversion here, returns the id

	return r.NodeID, nil
}

//ListNode implements modules.IdentityManager interface
func (g *Gedis) ListNode(farmID modules.Identifier, country string, city string) error {
	req := gedisListNodeBodyPayload{
		FarmID:  farmID.Identity(),
		Country: country,
		City:    city,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("nodes", "list", b)
	if err != nil {
		return parseError(err)
	}

	nl := &gedisListNodeResponseBody{}
	if err := json.Unmarshal(resp, &nl); err != nil {
		return err
	}

	// FIXME: gateway to list of types.Node
	fmt.Println(nl)

	return nil
}

//RegisterFarm implements modules.IdentityManager interface
func (g *Gedis) RegisterFarm(farm modules.Identifier, name string, email string, wallet []string) (string, error) {
	req := gedisRegisterFarmBody{
		Farm: gedisRegisterFarmBodyPayload{
			ThreebotID: farm.Identity(),
			Name:       name,
			Email:      email,
			Wallet:     wallet,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := g.sendCommand("farms", "register", b)
	if err != nil {
		fmt.Println("nope")
		return "", parseError(err)
	}

	fmt.Println(resp)

	r := tfgridFarm1{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}

	return r.Name, parseError(err)
}

//GetNode implements modules.IdentityManager interface
func (g *Gedis) GetNode(nodeID modules.Identifier) (*types.Node, error) {
	req := getNodeBody{
		NodeID: nodeID.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("nodes", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	n := tfgridNode2{}
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, err
	}
	return &types.Node{
		NodeID: n.NodeID,
		FarmID: n.FarmID,
	}, nil
}

func (g *Gedis) updateGenericNodeCapacity(captype string, node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	req := gedisNodeUpdateCapacity{
		NodeID: node.Identity(),
		Resource: tfgridNodeResource1{
			Mru: mru,
			Cru: cru,
			Hru: hru,
			Sru: sru,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommandBool("nodes", "update_"+captype+"_capacity", b)
	if err != nil {
		fmt.Println(err)
		return parseError(err)
	}

	return nil
}

//UpdateTotalNodeCapacity implements modules.IdentityManager interface
func (g *Gedis) UpdateTotalNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("total", node, mru, cru, hru, sru)
}

//UpdateReservedNodeCapacity implements modules.IdentityManager interface
func (g *Gedis) UpdateReservedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("reserved", node, mru, cru, hru, sru)
}

//UpdateUsedNodeCapacity implements modules.IdentityManager interface
func (g *Gedis) UpdateUsedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("used", node, mru, cru, hru, sru)
}

//GetFarm implements modules.IdentityManager interface
func (g *Gedis) GetFarm(farm modules.Identifier) (*network.Farm, error) {
	req := gedisGetFarmBody{
		Farm: tfgridFarm1{
			Name: farm.Identity(),
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(b))

	resp, err := g.sendCommand("farms", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	f := &network.Farm{}
	if err := json.Unmarshal(resp, &farm); err != nil {
		return nil, err
	}

	return f, nil
}

//UpdateFarm implements modules.IdentityManager interface
func (g *Gedis) UpdateFarm(farm modules.Identifier) error {
	req := gedisUpdateFarmBody{
		FarmID: farm.Identity(),
		Farm: tfgridFarm1{
			ThreebotID:      "",
			IyoOrganization: "",
			Name:            "",
			WalletAddresses: []string{},
			// Location: {},
			Vta: "",
			// ResourcePrice: {},
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommand("farms", "update", b)
	if err != nil {
		return parseError(err)
	}

	return nil
}

//ListFarm implements modules.IdentityManager interface
func (g *Gedis) ListFarm(country string, city string) error {
	req := gedisListFarmBody{
		Country: country,
		City:    city,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("farms", "list", b)
	if err != nil {
		return parseError(err)
	}

	fl := &gedisListFarmResponseBody{}
	if err := json.Unmarshal(resp, &fl); err != nil {
		return err
	}

	// FIXME: gateway to list of network.Farm
	fmt.Println(fl)

	return nil
}
