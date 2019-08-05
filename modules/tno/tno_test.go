package tno

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

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
	assert.Equal(t, uint16(1600), n.Resources[0].Peers[0].Connection.Port)
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
				port:     1600,
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
				port:     0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &modules.Network{}

			err = Configure(n, []Opts{
				AddNode(tt.args.nodeID, tt.args.farmID, tt.args.allocation, tt.args.key, tt.args.publicIP, tt.args.port),
			})
			assert.Error(t, err, "AddNode should return an error if the network does not have a PrefixZero configured")

			n.PrefixZero = &net.IPNet{
				IP:   net.ParseIP("2a02:1802:5e::"),
				Mask: net.CIDRMask(64, 128),
			}

			err = Configure(n, []Opts{
				AddNode(tt.args.nodeID, tt.args.farmID, tt.args.allocation, tt.args.key, tt.args.publicIP, tt.args.port),
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
