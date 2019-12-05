package gedis

import (
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

//UptimeUpdate send the uptime of the node to BCDB
func (g *Gedis) UptimeUpdate(nodeID pkg.Identifier, uptime uint64) error {

	_, err := g.Send("tfgrid.directory.nodes", "uptime_update", Args{
		"node_id": nodeID.Identity(),
		"uptime":  uptime,
	})

	return err
}

func (g *Gedis) updateGenericNodeCapacity(captype string, node pkg.Identifier, mru, cru, hru, sru uint64) error {
	_, err := g.Send("tfgrid.directory.nodes", "update_"+captype+"_capacity", Args{
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

// SendHardwareProof sends a dump of hardware and disks to BCDB
func (g *Gedis) SendHardwareProof(node pkg.Identifier, dmi interface{}, disks interface{}) error {
	_, err := g.Send("tfgrid.directory.nodes", "add_proof", Args{
		"node_id": node.Identity(),
		"proof": map[string]interface{}{
			"hardware": dmi,
			"disks":    disks,
		},
	})

	return err
}
