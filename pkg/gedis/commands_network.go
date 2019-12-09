package gedis

import (
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/schema"
)

func (g *Gedis) getLocalInterfaces(ifaces []types.IfaceInfo) []directory.TfgridNodeIface1 {
	output := make([]directory.TfgridNodeIface1, 0, len(ifaces))
	for _, iface := range ifaces {
		output = append(output,
			directory.TfgridNodeIface1{
				Name: iface.Name,
				Addrs: func() []schema.IPRange {
					var o []schema.IPRange
					for _, r := range iface.Addrs {
						o = append(o, r.ToSchema())
					}
					return o
				}(),
				Gateway: iface.Gateway,
			},
		)
	}

	return output
}

//PublishInterfaces implements network.TNoDB interface
func (g *Gedis) PublishInterfaces(local pkg.Identifier, ifaces []types.IfaceInfo) error {
	output := g.getLocalInterfaces(ifaces)

	_, err := g.Send("tfgrid.directory.nodes", "publish_interfaces", Args{
		"node_id": local.Identity(),
		"ifaces":  output,
	})

	return err
}

//GetPubIface gets public config of a node
func (g *Gedis) GetPubIface(node pkg.Identifier) (*types.PubIface, error) {
	object, err := g.GetNode(node)
	if err != nil {
		return nil, err
	}

	if object.PublicConfig == nil {
		return nil, network.ErrNoPubIface
	}

	return object.PublicConfig, nil
}

//SetPublicIface implements network.TNoDB interface
func (g *Gedis) SetPublicIface(node pkg.Identifier, pub *types.PubIface) error {
	public := directory.TfgridNodePublicIface1{
		Master:  pub.Master,
		Type:    directory.TfgridNodePublicIface1TypeMacvlan,
		Version: int64(pub.Version),
	}

	if !pub.IPv4.Nil() {
		public.Ipv4 = pub.IPv4.ToSchema()
		public.Gw4 = pub.GW4
	}

	if !pub.IPv6.Nil() {
		public.Ipv6 = pub.IPv6.ToSchema()
		public.Gw6 = pub.GW6
	}

	_, err := g.Send("tfgrid.directory.nodes", "set_public_iface", Args{
		"node_id": node.Identity(),
		"public":  public,
	})

	return err
}

//PublishWGPort implements network.TNoDB interface
func (g *Gedis) PublishWGPort(node pkg.Identifier, ports []uint) error {
	_, err := g.Send("tfgrid.directory.nodes", "publish_wg_ports", Args{
		"node_id": node.Identity(),
		"ports":   ports,
	})

	return err
}
