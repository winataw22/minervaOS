package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkUnmarshal(t *testing.T) {
	input := `{
		"network_id": "netid",
		"resources": [
			{
				"node_id": {
					"id": "kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=",
					"farmer_id": "ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E=",
					"reachability_v4": "public",
					"reachability_v6": "public"
				},
				"prefix": "2a02:2788:864:1314:9eb6::/64",
				"link_local": "fe80::9eb6:d0ff:fe97:764b/64",
				"peers": [
					{
						"type": "wireguard",
						"prefix": "2a02:2788:864:1314:9eb6::/64",
						"connection": {
							"ip": "2a02:2788:864:1314:9eb6::1/64",
							"port": 1600,
							"key": "X9A2VGvJZT/mYGMXWd4BXFskfziPLraYSgdpIGUgmm0="
						}
					}
				]
			}
		],
		"exit_point": {
			"node_id": "kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=",
			"prefix": "2a02:2788:864:1314:9eb6::/64",
			"link_local": "fe80::9eb6:d0ff:fe97:764b/64",
			"peers": [
				{
					"type": "wireguard",
					"prefix": "2a02:2788:864:1314:9eb6::/64",
					"connection": {
						"ip": "2a02:2788:864:1314:9eb6::1/64",
						"port": 1600,
						"key": "X9A2VGvJZT/mYGMXWd4BXFskfziPLraYSgdpIGUgmm0="
					}
				}
			],
			"ipv4_conf": {
				"cidr": "192.168.0.1/24",
				"gateway": "192.168.0.254",
				"metric": 302,
				"iface": "eth0",
				"enable_nat": true
			},
			"ipv4_dnat": [{
				"internal_ip": "192.168.0.1",
				"internal_port": 80,
				"external_ip": "172.20.0.14",
				"external_port": 8080,
				"protocol": "tcp"
			}],
			"ipv6_conf": {
				"addr": "2a02:2788:864:1314:9eb6:d0ff:fe97:764b/64",
				"gateway": "2a02:2788:864:1314:9eb6:d0ff:fe97:1",
				"metric": 301,
				"iface": "wlna0"
			}
		}
	}`

	r := strings.NewReader(input)
	network := Network{}
	err := json.NewDecoder(r).Decode(&network)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Equal(NetID("netid"), network.NetID)
	assert.Equal(1, len(network.Resources))
	assert.Equal("kV3u7GJKWA7Js32LmNA5+G3A0WWnUG9h+5gnL6kr6lA=", network.Resources[0].NodeID.ID)
	assert.Equal("ZF6jtCblLhTgAqp2jvxKkOxBgSSIlrRh1mRGiZaRr7E=", network.Resources[0].NodeID.FarmerID)
}

func TestEncodeDecode(t *testing.T) {
	network := &Network{
		NetID: NetID("test"),
	}
	b, err := json.Marshal(network)
	require.NoError(t, err)
	fmt.Println(string(b))

	decoded := Network{}
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)
	assert.Equal(t, network, decoded)
}
