package pkg

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func TestNetResourceUnmarshal(t *testing.T) {
	input := `{
		"name": "testnetworkd",
		"net_id": "net1",
		"ip_range": "10.0.0.0/16",
		"node_id": "HAcDwf7oCWEbn7ME1W4j3ACfsUo5kUgJqhk5MEDkbKis",
		"subnet": "10.0.1.0/24",
		"wg_private_key": "988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d",
		"wg_public_key": "L+V9o0fNYkMVKNqsX7spBzD/9oSvxM/C7ZCZX1jLO3Q=",
		"wg_listen_port": 6380
	}`

	r := strings.NewReader(input)
	network := NetResource{}
	err := json.NewDecoder(r).Decode(&network)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Equal(NetID("net1"), network.NetID)
	assert.Equal("HAcDwf7oCWEbn7ME1W4j3ACfsUo5kUgJqhk5MEDkbKis", network.NodeID)
	assert.Equal("988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d", network.WGPrivateKey)
	assert.Equal("L+V9o0fNYkMVKNqsX7spBzD/9oSvxM/C7ZCZX1jLO3Q=", network.WGPublicKey)
	assert.Equal("10.0.1.0/24", network.Subnet.String())

}

func TestEncodeDecode(t *testing.T) {
	network := &NetResource{
		NetID: NetID("test"),
		Name:  "supernet",
		NetworkIPRange: types.NewIPNet(&net.IPNet{
			IP:   net.ParseIP("10.0.0.0"),
			Mask: net.CIDRMask(16, 32),
		}),
		NodeID: "node1",
		Subnet: types.NewIPNet(&net.IPNet{
			IP:   net.ParseIP("10.0.1.0"),
			Mask: net.CIDRMask(24, 32),
		}),
		Peers: []Peer{
			{
				Subnet: types.NewIPNet(&net.IPNet{
					IP:   net.ParseIP("10.0.2.0"),
					Mask: net.CIDRMask(24, 32),
				}),
				Endpoint:    "172.20.0.90:6380",
				WGPublicKey: "pubkey",
				AllowedIPs: []types.IPNet{
					types.NewIPNet(&net.IPNet{
						IP:   net.ParseIP("10.0.1.0"),
						Mask: net.CIDRMask(24, 32),
					}),
				},
			},
		},
	}
	b, err := json.Marshal(network)
	require.NoError(t, err)
	fmt.Println(string(b))

	decoded := &NetResource{}
	err = json.Unmarshal(b, decoded)
	require.NoError(t, err)
	assert.Equal(t, network.Name, decoded.Name)
	assert.Equal(t, network.NetworkIPRange.String(), decoded.NetworkIPRange.String())
	assert.Equal(t, network.NetID, decoded.NetID)

	assert.Equal(t, network.NodeID, decoded.NodeID)
	assert.Equal(t, network.Subnet.String(), decoded.Subnet.String())
	assert.Equal(t, network.WGPrivateKey, decoded.WGPrivateKey)
	assert.Equal(t, network.WGPublicKey, decoded.WGPublicKey)
	assert.Equal(t, network.WGListenPort, decoded.WGListenPort)
	require.Equal(t, len(network.Peers), len(decoded.Peers))

	ePeer := network.Peers[0]
	aPeer := decoded.Peers[0]

	assert.Equal(t, ePeer.Subnet.String(), aPeer.Subnet.String())
	assert.Equal(t, ePeer.Endpoint, aPeer.Endpoint)
	assert.Equal(t, ePeer.AllowedIPs, aPeer.AllowedIPs)
	assert.Equal(t, ePeer.WGPublicKey, aPeer.WGPublicKey)
}
