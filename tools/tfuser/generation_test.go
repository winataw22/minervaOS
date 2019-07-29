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
			IP:         net.ParseIP("2001:1:1:1::1"),
			Port:       1601,
			Key:        "MAX+6SN7sw5GedMEUcHMbPiaiA0OFKGfa86+O9Pc014=",
			PrivateKey: "88edc0808fdb952ce570cb6d40046a38a2cbd22cc05500175be88fc1285cd077f088a4dc366d5023387f380e05455f088bf6af74e22bd02682ae303b326f5d7e1a1318be38132c1e9c0efe577c45692beaffec33015acff5a237399c",
		},
	},
	{
		Type:   modules.ConnTypeWireguard,
		Prefix: mustParseCIDR("2a02:1802:5e:cc02::/64"),
		Connection: modules.Wireguard{
			IP:         net.ParseIP("2001:1:1:2::1"),
			Port:       1602,
			Key:        "XhxfA3tJG9VwMRiLGdyVmPb2B+u54lUsJRik4D8Cv2Q=",
			PrivateKey: "91237febe891e7c797961540bde404440ae2aab7913433a53725f1aea13bfb65248fc8fe4f733a78c056e290e1878c2833dc3ca063dec4a4ec1d0ef6ed7ad0def3b425bb9b385752e5189b919f2b77d5f3439cce4701cc1469c9e366",
		},
	},
	{
		Type:   modules.ConnTypeWireguard,
		Prefix: mustParseCIDR("2a02:1802:5e:aaaa::/64"),
		Connection: modules.Wireguard{
			IP:         net.ParseIP("2001:3:3:3::3"),
			Port:       1603,
			Key:        "QaaAugFQYVs7Hr/FnPUNZ2aWem/tnRB1IZ2lhnBt6Gg=",
			PrivateKey: "5840f09c3ed6e1785dd451f5687b9692d6d2bfaab0a60173adea73d60fc40b7773f7c6a01f284fc4ce456472010703929e06abbc3584eb7cd84ccaa2f5e3fb22c27601a60174202cf8e5dd28b49e848e8fc589cbcd061e456613dc5e",
		},
	},
}

var node1 = &modules.NetResource{
	NodeID: &modules.NodeID{
		ID:             "qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3",
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
		ID:             "DLFF6CAshvyhCrpyTHq1dMd6QP6kFyhrVGegTgudk6xk",
		ReachabilityV4: modules.ReachabilityV4Hidden,
		ReachabilityV6: modules.ReachabilityV6ULA,
	},
	Prefix:    mustParseCIDR("2a02:1802:5e:cc02::/64"),
	LinkLocal: mustParseCIDR("fe80::cc02/64"),
	Peers:     peers,
}

var node3 = &modules.NetResource{
	NodeID: &modules.NodeID{
		ID:             "37zg5cmfHQdMmzcqdBR7YFRCQZqA35wdEChk7ccR4tNM",
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
	out, err := genWGQuick(testNetwork, "37zg5cmfHQdMmzcqdBR7YFRCQZqA35wdEChk7ccR4tNM", "privatekey")
	require.NoError(t, err)
	assert.EqualValues(t, `
[Interface]
PrivateKey = privatekey
Address = 2a02:1802:5e:aaaa::/64, fe80::aaaa/64, 10.255.170.170/16


[Peer]
PublicKey = MAX&#43;6SN7sw5GedMEUcHMbPiaiA0OFKGfa86&#43;O9Pc014=
AllowedIPs = fe80::1/128, 10.0.0.0/8, 2a02:1802:5e::/48
PersistentKeepalive = 20
Endpoint = [2001:1:1:1::1]:1601
`, out)
}
