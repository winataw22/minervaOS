package gedis

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/gedis/types/directory"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/schema"
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
						o = append(o, schema.IPRange{IPNet: *r})
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
func (g *Gedis) PublishInterfaces(local modules.Identifier, ifaces []types.IfaceInfo) error {
	output := g.getLocalInterfaces(ifaces)

	_, err := g.Send("nodes", "publish_interfaces", Args{
		"node_id": local.Identity(),
		"ifaces":  output,
	})

	return err
}

//GetPubIface gets public config of a node
func (g *Gedis) GetPubIface(node modules.Identifier) (*types.PubIface, error) {
	object, err := g.GetNode(node)
	if err != nil {
		return nil, err
	}

	if object.PublicConfig == nil {
		return nil, fmt.Errorf("public config not set")
	}

	return object.PublicConfig, nil
}

//SetPublicIface implements network.TNoDB interface
func (g *Gedis) SetPublicIface(node modules.Identifier, pub *types.PubIface) error {
	public := directory.TfgridNodePublicIface1{
		Master:  pub.Master,
		Type:    directory.TfgridNodePublicIface1TypeMacvlan,
		Version: int64(pub.Version),
	}

	if pub.IPv4 != nil {
		public.Ipv4 = schema.IPRange{IPNet: *pub.IPv4}
		public.Gw4 = pub.GW4
	}

	if pub.IPv6 != nil {
		public.Ipv6 = schema.IPRange{IPNet: *pub.IPv6}
		public.Gw6 = pub.GW6
	}

	_, err := g.Send("nodes", "set_public_iface", Args{
		"node_id": node.Identity(),
		"public":  public,
	})

	return err
}

//PublishWGPort implements network.TNoDB interface
func (g *Gedis) PublishWGPort(node modules.Identifier, ports []uint) error {
	_, err := g.Send("nodes", "publish_wg_ports", Args{
		"node_id": node.Identity(),
		"ports":   ports,
	})

	return err
}
