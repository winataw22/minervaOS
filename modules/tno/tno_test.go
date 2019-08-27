package tno

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func mustParseCIDR(cidr string) *net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return ipnet
}

var peers = []*modules.Peer{
	{
		Type:   modules.ConnTypeWireguard,
		Prefix: mustParseCIDR("2a02:1802:5e:ff02::/64"),
		Connection: modules.Wireguard{
			IP:   net.ParseIP("2001:1:1:1::1"),
			Port: 1601,
			Key:  "4w4woC+AuDUAaRipT49M8SmTkzERps3xA5i0BW4hPiw=",
		},
	},
	{
		Type:   modules.ConnTypeWireguard,
		Prefix: mustParseCIDR("2a02:1802:5e:cc02::/64"),
		Connection: modules.Wireguard{
			IP:   net.ParseIP("2001:1:1:2::1"),
			Port: 1602,
			Key:  "HXnTmizQdGlAuE9PpVPw1Drg2WygUsxwGnJY+A5xgVo=",
		},
	},
	{
		Type:   modules.ConnTypeWireguard,
		Prefix: mustParseCIDR("2a02:1802:5e:aaaa::/64"),
		Connection: modules.Wireguard{
			IP:   net.ParseIP("2001:3:3:3::3"),
			Port: 1603,
			Key:  "5Adc456lkjlRtRipT49M8SmTkzERps3xA5i0BW4hPiw=",
		},
	},
}

var node1 = &modules.NetResource{
	NodeID: &modules.NodeID{
		ID:             "node1",
		ReachabilityV4: modules.ReachabilityV4Public,
		ReachabilityV6: modules.ReachabilityV6Public,
	},
	Prefix:    mustParseCIDR("2a02:1802:5e:ff02::/64"),
	LinkLocal: mustParseCIDR("fe80::ff02/64"),
	Peers:     peers,
	ExitPoint: true,
}
var node2 = &modules.NetResource{
	NodeID: &modules.NodeID{
		ID:             "node2",
		ReachabilityV4: modules.ReachabilityV4Hidden,
		ReachabilityV6: modules.ReachabilityV6ULA,
	},
	Prefix:    mustParseCIDR("2a02:1802:5e:cc02::/64"),
	LinkLocal: mustParseCIDR("fe80::cc02/64"),
	Peers:     peers,
}

var node3 = &modules.NetResource{
	NodeID: &modules.NodeID{
		ID:             "node3",
		ReachabilityV4: modules.ReachabilityV4Public,
		ReachabilityV6: modules.ReachabilityV6Public,
	},
	Prefix:    mustParseCIDR("2a02:1802:5e:aaaa::/64"),
	LinkLocal: mustParseCIDR("fe80::aaaa/64"),
	Peers:     peers,
}

var networks = []*modules.Network{
	{
		NetID: "net1",
		Resources: []*modules.NetResource{
			node1, node2, node3,
		},
		PrefixZero: mustParseCIDR("2a02:1802:5e:0000::/64"),
		Exit:       &modules.ExitPoint{},
	},
}

func TestGenerateID(t *testing.T) {
	n := &modules.Network{}

	err := Configure(n, []Opts{
		GenerateID(),
	})
	require.NoError(t, err)
	assert.NotEqual(t, "", string(n.NetID))
}

func TestConfigurePrefixZero(t *testing.T) {
	n := &modules.Network{}

	err := Configure(n, []Opts{
		ConfigurePrefixZero(&net.IPNet{
			IP:   net.ParseIP("2a02:1802:5e::"),
			Mask: net.CIDRMask(48, 128),
		}),
	})
	require.NoError(t, err)

	assert.Equal(t, &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e::"),
		Mask: net.CIDRMask(64, 128),
	}, n.PrefixZero)
}

func TestConfigureExitResource(t *testing.T) {
	n := &modules.Network{
		PrefixZero: &net.IPNet{
			IP:   net.ParseIP("2a02:1802:5e::"),
			Mask: net.CIDRMask(64, 128),
		},
	}
	allocation := &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e:afba::"),
		Mask: net.CIDRMask(64, 128),
	}

	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	publicIP := net.ParseIP("2a02:1802:5e::223")

	err = Configure(n, []Opts{
		ConfigureExitResource("DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk", allocation, publicIP, key, 48),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(n.Resources))
	assert.Equal(t, "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk", n.Resources[0].NodeID.ID)
	// assert.Equal(t, "", n.Resources[0].NodeID.FarmerID)
	assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Prefix.String())
	assert.Equal(t, "fe80::afba/64", n.Resources[0].LinkLocal.String())
	assert.True(t, n.Resources[0].ExitPoint)
	assert.Equal(t, 1, len(n.Resources[0].Peers))
	assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Peers[0].Prefix.String())
	assert.Equal(t, modules.ConnTypeWireguard, n.Resources[0].Peers[0].Type)
	assert.Equal(t, "2a02:1802:5e::223", n.Resources[0].Peers[0].Connection.IP.String())
	assert.Equal(t, uint16(44986), n.Resources[0].Peers[0].Connection.Port)
	assert.Equal(t, key.PublicKey().String(), n.Resources[0].Peers[0].Connection.Key)

	assert.NotNil(t, n.Exit)
	assert.NotNil(t, n.Exit.Ipv6Conf)
	assert.Equal(t, "fe80::afba/64", n.Exit.Ipv6Conf.Addr.String())
	assert.Equal(t, "fe80::1", n.Exit.Ipv6Conf.Gateway.String())
	assert.Equal(t, "public", n.Exit.Ipv6Conf.Iface)
}

func TestAddNode(t *testing.T) {
	type args struct {
		nodeID     string
		farmID     string
		allocation *net.IPNet
		key        wgtypes.Key
		publicIP   net.IP
		port       uint16
	}

	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name string
		args args
	}{
		{
			name: "public",
			args: args{
				nodeID: "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk",
				farmID: "7koUE4nRbdsqEbtUVBhx3qvRqF58gfeHGMRGJxjqwfZi",
				allocation: &net.IPNet{
					IP:   net.ParseIP("2a02:1802:5e:afba::"),
					Mask: net.CIDRMask(64, 128),
				},
				key:      key,
				publicIP: net.ParseIP("2a02:1802:5e::afba"),
				port:     44986,
			},
		},
		{
			name: "private",
			args: args{
				nodeID: "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk",
				farmID: "7koUE4nRbdsqEbtUVBhx3qvRqF58gfeHGMRGJxjqwfZi",
				allocation: &net.IPNet{
					IP:   net.ParseIP("2a02:1802:5e:afba::"),
					Mask: net.CIDRMask(64, 128),
				},
				key:      key,
				publicIP: nil,
				port:     44986,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &modules.Network{}

			err = Configure(n, []Opts{
				AddNode(tt.args.nodeID, tt.args.farmID, tt.args.allocation, tt.args.key, tt.args.publicIP),
			})
			assert.Error(t, err, "AddNode should return an error if the network does not have a PrefixZero configured")

			n.PrefixZero = &net.IPNet{
				IP:   net.ParseIP("2a02:1802:5e::"),
				Mask: net.CIDRMask(64, 128),
			}

			err = Configure(n, []Opts{
				AddNode(tt.args.nodeID, tt.args.farmID, tt.args.allocation, tt.args.key, tt.args.publicIP),
			})

			require.NoError(t, err)
			assert.Equal(t, 1, len(n.Resources))
			assert.Equal(t, tt.args.nodeID, n.Resources[0].NodeID.ID)
			// assert.Equal(t, "", n.Resources[0].NodeID.FarmerID)
			assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Prefix.String())
			assert.Equal(t, "fe80::afba/64", n.Resources[0].LinkLocal.String())
			assert.False(t, n.Resources[0].ExitPoint)
			assert.Equal(t, modules.ReachabilityV4Hidden, n.Resources[0].NodeID.ReachabilityV4)
			if tt.args.publicIP != nil && tt.args.port != 0 {
				assert.Equal(t, modules.ReachabilityV6Public, n.Resources[0].NodeID.ReachabilityV6)
			} else {
				assert.Equal(t, modules.ReachabilityV6ULA, n.Resources[0].NodeID.ReachabilityV6)
			}
			assert.Equal(t, 1, len(n.Resources[0].Peers))
			assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Peers[0].Prefix.String())
			assert.Equal(t, modules.ConnTypeWireguard, n.Resources[0].Peers[0].Type)
			assert.Equal(t, tt.args.publicIP, n.Resources[0].Peers[0].Connection.IP)
			assert.Equal(t, tt.args.port, n.Resources[0].Peers[0].Connection.Port)
			assert.Equal(t, tt.args.key.PublicKey().String(), n.Resources[0].Peers[0].Connection.Key)
		})
	}
}

func TestAddUser(t *testing.T) {
	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	allocation := &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e:afba::"),
		Mask: net.CIDRMask(64, 128),
	}

	n := &modules.Network{}

	err = Configure(n, []Opts{
		AddUser("DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk", allocation, key),
	})
	assert.Error(t, err, "AddUser should return an error if the network does not have a PrefixZero configured")

	n.PrefixZero = &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e::"),
		Mask: net.CIDRMask(64, 128),
	}

	err = Configure(n, []Opts{
		AddUser("DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk", allocation, key),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(n.Resources))
	assert.Equal(t, "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk", n.Resources[0].NodeID.ID)
	assert.Equal(t, modules.ReachabilityV6ULA, n.Resources[0].NodeID.ReachabilityV6)
	assert.Equal(t, modules.ReachabilityV4Hidden, n.Resources[0].NodeID.ReachabilityV4)
	// assert.Equal(t, "", n.Resources[0].NodeID.FarmerID)
	assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Prefix.String())
	assert.Equal(t, "fe80::afba/64", n.Resources[0].LinkLocal.String())
	assert.False(t, n.Resources[0].ExitPoint)
	require.Equal(t, 1, len(n.Resources[0].Peers))
	assert.Equal(t, "2a02:1802:5e:afba::/64", n.Resources[0].Peers[0].Prefix.String())
	assert.Equal(t, modules.ConnTypeWireguard, n.Resources[0].Peers[0].Type)
	assert.Nil(t, n.Resources[0].Peers[0].Connection.IP)
	assert.Zero(t, n.Resources[0].Peers[0].Connection.Port)
	assert.Equal(t, key.PublicKey().String(), n.Resources[0].Peers[0].Connection.Key)
}

func TestRemoveNode(t *testing.T) {
	n := networks[0]

	assert.EqualValues(t, node1, n.Resources[0])
	assert.EqualValues(t, node2, n.Resources[1])
	assert.EqualValues(t, node3, n.Resources[2])

	err := Configure(n, []Opts{
		RemoveNode(node1.NodeID.ID),
	})
	require.NoError(t, err)

	assert.EqualValues(t, node2, n.Resources[0])
	assert.EqualValues(t, node3, n.Resources[1])
}
