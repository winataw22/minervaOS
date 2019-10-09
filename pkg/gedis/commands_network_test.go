package gedis

import (
	"net"
	"testing"

	"github.com/threefoldtech/zos/pkg/schema"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func TestNetworkPublishInterfaces(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := pkg.StrIdentifier("node-1")
	r := schema.MustParseIPRange("192.168.1.2/24")
	args := Args{
		"node_id": id,
		"ifaces": []directory.TfgridNodeIface1{
			{
				Name: "eth0",
				Addrs: []schema.IPRange{
					r,
				},
				Gateway: []net.IP{
					net.ParseIP("192.168.1.1"),
				},
			},
		},
	}

	conn.On("Do", "default.nodes.publish_interfaces", mustMarshal(t, args)).
		Return(nil, nil)

	inf := types.IfaceInfo{
		Name: "eth0",
		Addrs: []*net.IPNet{
			&r.IPNet,
		},
		Gateway: []net.IP{net.ParseIP("192.168.1.1")},
	}
	err := gedis.PublishInterfaces(id, []types.IfaceInfo{inf})

	require.NoError(err)
	conn.AssertCalled(t, "Close")
}

func TestNetworkSetPublicIface(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := pkg.StrIdentifier("node-1")
	r := schema.MustParseIPRange("192.168.1.2/24")
	args := Args{
		"node_id": id,
		"public": directory.TfgridNodePublicIface1{
			Master: "eth0",
			Ipv4:   r,
			Gw4:    net.ParseIP("192.168.1.1"),
		},
	}

	conn.On("Do", "default.nodes.set_public_iface", mustMarshal(t, args)).
		Return(nil, nil)

	err := gedis.SetPublicIface(id, &types.PubIface{
		Master: "eth0",
		Type:   types.MacVlanIface,
		IPv4:   &r.IPNet,
		GW4:    net.ParseIP("192.168.1.1"),
	})

	require.NoError(err)
	conn.AssertCalled(t, "Close")
}
