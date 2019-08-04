package network

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"

	"github.com/threefoldtech/zosv2/modules/identity"

	"golang.org/x/crypto/ed25519"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zosv2/modules/crypto"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

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

func TestCreateNetwork(t *testing.T) {
	var (
		network    = networks[0]
		resource   = network.Resources[0]
		nibble     = zosip.NewNibble(resource.Prefix, network.AllocationNR)
		netName    = nibble.NetworkName()
		bridgeName = nibble.BridgeName()
		vethName   = nibble.VethName()
	)

	dir, err := ioutil.TempDir("", netName)
	require.NoError(t, err)

	idMgr, err := identity.NewManager(filepath.Join(dir, "id"))
	require.NoError(t, err)

	storage := filepath.Join(dir, netName)
	networker := &networker{
		identity:   idMgr,
		storageDir: storage,
	}

	for _, tc := range []struct {
		exitIface *PubIface
	}{
		{
			exitIface: nil,
		},
		{
			exitIface: &PubIface{
				Master: "zos0",
				Type:   MacVlanIface,
				IPv6:   mustParseCIDR("2a02:1802:5e:ff02::100/64"),
				GW6:    net.ParseIP("fe80::1"),
			},
		},
	} {
		name := "withPublicNamespace"
		if tc.exitIface == nil {
			name = "NoPubNamespace"
		}
		t.Run(name, func(t *testing.T) {
			defer func() {
				err := networker.DeleteNetResource(*network)
				require.NoError(t, err)
				if tc.exitIface != nil {
					pubNs, _ := namespace.GetByName(PublicNamespace)
					err = namespace.Delete(pubNs)
					require.NoError(t, err)
				}
			}()

			if tc.exitIface != nil {
				err := CreatePublicNS(tc.exitIface)
				require.NoError(t, err)
			}

			err := createNetworkResource(resource, network)
			require.NoError(t, err)

			assert.True(t, bridge.Exists(bridgeName))
			assert.True(t, namespace.Exists(netName))

			netns, err := namespace.GetByName(netName)
			require.NoError(t, err)
			defer netns.Close()
			var handler = func(_ ns.NetNS) error {
				link, err := netlink.LinkByName(vethName)
				require.NoError(t, err)

				addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
				require.NoError(t, err)
				assert.Equal(t, 1, len(addrs))
				assert.Equal(t, "10.255.2.1/24", addrs[0].IPNet.String())

				addrs, err = netlink.AddrList(link, netlink.FAMILY_V6)
				require.NoError(t, err)
				assert.Equal(t, 2, len(addrs))
				assert.Equal(t, "2a02:1802:5e:ff02::/64", addrs[0].IPNet.String())

				return nil
			}
			err = netns.Do(handler)
			assert.NoError(t, err)
		})
	}

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
