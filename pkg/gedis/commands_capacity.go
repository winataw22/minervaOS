package gedis

import (
	"github.com/threefoldtech/zos/pkg"
)

//UptimeUpdate send the uptime of the node to BCDB
func (g *Gedis) UptimeUpdate(nodeID pkg.Identifier, uptime uint64) error {

	_, err := g.Send("nodes", "uptime_update", Args{
		"node_id": nodeID.Identity(),
		"uptime":  uptime,
	})

	return err
}
