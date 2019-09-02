package nr

import (
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	"golang.org/x/crypto/ed25519"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zosv2/modules/crypto"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

type testIdentityManager struct {
	id   string
	farm string
}

var _ modules.IdentityManager = (*testIdentityManager)(nil)

// NodeID returns the node id (public key)
func (t *testIdentityManager) NodeID() modules.StrIdentifier {
	return modules.StrIdentifier(t.id)
}

// FarmID return the farm id this node is part of. this is usually a configuration
// that the node is booted with. An error is returned if the farmer id is not configured
func (t *testIdentityManager) FarmID() (modules.StrIdentifier, error) {
	return modules.StrIdentifier(t.farm), nil
}

// Sign signs the message with privateKey and returns a signature.
func (t *testIdentityManager) Sign(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Verify reports whether sig is a valid signature of message by publicKey.
func (t *testIdentityManager) Verify(message, sig []byte) error {
	return fmt.Errorf("not implemented")
}

// Encrypt encrypts message with the public key of the node
func (t *testIdentityManager) Encrypt(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Decrypt decrypts message with the private of the node
func (t *testIdentityManager) Decrypt(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
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
	ExitPoint: 1,
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

func TestCreate(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	network := networks[0]
	resource := network.Resources[0]
	nibble, err := zosip.NewNibble(resource.Prefix, network.AllocationNR)
	require.NoError(err)
	bridgeName := nibble.BridgeName()
	vethName := nibble.NRLocalName()

	netRes, err := New(resource.NodeID.ID, network, wgtypes.Key{})
	require.NoError(err)

	defer func() {
		err := netRes.Delete()
		require.NoError(err)
	}()

	err = netRes.Create(nil) //TODO test with a public namespace
	require.NoError(err)

	assert.True(bridge.Exists(bridgeName))
	assert.True(namespace.Exists(netRes.NamespaceName()))

	netns, err := namespace.GetByName(netRes.NamespaceName())
	require.NoError(err)
	defer netns.Close()
	var handler = func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(vethName)
		require.NoError(err)

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		require.NoError(err)
		assert.Equal(1, len(addrs))
		assert.Equal("10.255.2.1/24", addrs[0].IPNet.String())

		addrs, err = netlink.AddrList(link, netlink.FAMILY_V6)
		require.NoError(err)
		assert.Equal(2, len(addrs))
		assert.Equal("2a02:1802:5e:ff02::/64", addrs[0].IPNet.String())

		return nil
	}
	err = netns.Do(handler)
	assert.NoError(err)
}

func TestWGPeers(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	t.Run("we are exit peer", func(t *testing.T) {
		network := networks[0]
		resource := network.Resources[0]
		netRes, err := New(resource.NodeID.ID, network, wgtypes.Key{})
		require.NoError(err)

		actualPeers, err := netRes.wgPeers()
		require.NoError(err)

		assert.Equal([]*wireguard.Peer{
			{
				PublicKey: peers[1].Connection.Key,
				Endpoint:  "",
				AllowedIPs: []string{
					"fe80::cc02/128",
					"10.255.204.2/16",
					"2a02:1802:5e:cc02::/64",
				},
			},

			{
				PublicKey: peers[2].Connection.Key,
				Endpoint:  "[2001:3:3:3::3]:1603",
				AllowedIPs: []string{
					"fe80::aaaa/128",
					"10.255.170.170/16",
					"2a02:1802:5e:aaaa::/64",
				},
			},
		}, actualPeers)
	})

	t.Run("we are not exit peer", func(t *testing.T) {
		network := networks[0]
		resource := network.Resources[1]
		netRes, err := New(resource.NodeID.ID, network, wgtypes.Key{})
		require.NoError(err)

		actualPeers, err := netRes.wgPeers()
		require.NoError(err)

		assert.Equal([]*wireguard.Peer{
			{
				PublicKey: peers[0].Connection.Key,
				Endpoint:  "[2001:1:1:1::1]:1601",
				AllowedIPs: []string{
					"0.0.0.0/0",
					"::/0",
				},
			},
			{
				PublicKey: peers[2].Connection.Key,
				Endpoint:  "[2001:3:3:3::3]:1603",
				AllowedIPs: []string{
					"fe80::aaaa/128",
					"10.255.170.170/16",
					"2a02:1802:5e:aaaa::/64",
				},
			},
		}, actualPeers)
	})

}
func TestRoutes(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	t.Run("we are exit peer", func(t *testing.T) {
		network := networks[0]
		resource := network.Resources[0]
		netRes, err := New(resource.NodeID.ID, network, wgtypes.Key{})
		require.NoError(err)

		actualRoutes, err := netRes.routes()
		require.NoError(err)
		assert.Equal([]*netlink.Route{
			{
				Dst:       mustParseCIDR("2a02:1802:5e:cc02::/64"),
				Gw:        net.ParseIP("fe80::cc02"),
				LinkIndex: 0,
			},
			{
				Dst:       mustParseCIDR("2a02:1802:5e:aaaa::/64"),
				Gw:        net.ParseIP("fe80::aaaa"),
				LinkIndex: 0,
			},
		}, actualRoutes)
	})

	t.Run("we are not exit peer", func(t *testing.T) {
		network := networks[0]
		resource := network.Resources[1]
		netRes, err := New(resource.NodeID.ID, network, wgtypes.Key{})
		require.NoError(err)

		actualRoutes, err := netRes.routes()
		require.NoError(err)
		assert.Equal([]*netlink.Route{
			{
				Dst:       mustParseCIDR("0::/0"),
				Gw:        net.ParseIP("fe80::ff02"),
				LinkIndex: 0,
			},
			{
				Dst:       mustParseCIDR("10.255.2.0/24"),
				Gw:        net.ParseIP("10.255.255.2"),
				LinkIndex: 0,
			},
			{
				Dst:       mustParseCIDR("0.0.0.0/0"),
				Gw:        net.ParseIP("10.255.255.2"),
				LinkIndex: 0,
			},
			{
				Dst:       mustParseCIDR("2a02:1802:5e:aaaa::/64"),
				Gw:        net.ParseIP("fe80::aaaa"),
				LinkIndex: 0,
			},
		}, actualRoutes)
	})
}

func mustParseCIDR(cidr string) *net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return ipnet
}

func TestKeys(t *testing.T) {
	wgKey, err := wgtypes.GenerateKey()
	require.NoError(t, err)

	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	fmt.Println(wgKey.String())

	encrypted, err := crypto.Encrypt([]byte(wgKey.String()), pk)
	require.NoError(t, err)

	strEncrypted := fmt.Sprintf("%x", encrypted)

	strDecrypted := ""
	fmt.Sscanf(strEncrypted, "%x", &strDecrypted)

	decrypted, err := crypto.Decrypt([]byte(strDecrypted), sk)
	require.NoError(t, err)

	fmt.Println(string(decrypted))

	wgKey2, err := wgtypes.ParseKey(string(decrypted))
	require.NoError(t, err)

	assert.Equal(t, wgKey, wgKey2)
}
