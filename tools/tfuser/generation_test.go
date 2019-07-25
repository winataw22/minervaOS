package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
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

var testNetwork = &modules.Network{
	NetID: "net1",
	Resources: []*modules.NetResource{
		node1, node2, node3,
	},
	PrefixZero: mustParseCIDR("2a02:1802:5e:0000::/64"),
	Exit:       &modules.ExitPoint{},
}

func TestGenWGQuick(t *testing.T) {
	out, err := genWGQuick(testNetwork, "node3", "privatekey")
	require.NoError(t, err)
	assert.EqualValues(t, `
[Interface]
PrivateKey = privatekey
Address = 2a02:1802:5e:aaaa::/64, fe80::aaaa/64, 10.255.170.170/16


[Peer]
PublicKey = 4w4woC+AuDUAaRipT49M8SmTkzERps3xA5i0BW4hPiw=
AllowedIPs = fe80::1/128, 10.0.0.0/8, 2a02:1802:5e::/48
PersistentKeepalive = 20
Endpoint = [2001:1:1:1::1]:1601
`, out)
}
