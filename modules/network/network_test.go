package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

func TestInterfaces(t *testing.T) {
	link, err := netlink.LinkByName("wlan0")
	require.NoError(t, err)

	routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	require.NoError(t, err)

	for _, route := range routes {
		fmt.Printf("%v\n", route)
	}
}

var networks = []modules.Network{
	{
		NetID: "net1",
		Resources: []modules.NetResource{
			{
				NodeID:    modules.NodeID{ID: "node1"},
				Prefix:    mustParseCIDR("2a02:1802:5e:ff02::/64"),
				LinkLocal: mustParseCIDR("fe80::ff02/64"),
				Connected: []modules.Connected{
					{
						Type:   modules.ConnTypeWireguard,
						Prefix: mustParseCIDR("2a02:1802:5e:cc02::/64"),
						Connection: modules.Wireguard{
							IP:   net.ParseIP("2001:1:1:2::1"),
							Port: 1602,
							Key:  "4w4woC+AuDUAaRipT49M8SmTkzERps3xA5i0BW4hPiw=",
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
				},
			},
		},
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

	defer func() {
		_ = deleteNetResource(&resource, network.AllocationNR)
	}()

	err := createNetworkResource(network.NetID, &resource, network.AllocationNR)
	require.NoError(t, err)

	assert.True(t, bridge.Exists(bridgeName))
	assert.True(t, namespace.Exists(netName))

	netns, err := namespace.GetByName(netName)
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

}

func TestConfigureWG(t *testing.T) {
	var (
		network  = networks[0]
		resource = network.Resources[0]
		nibble   = zosip.NewNibble(resource.Prefix, network.AllocationNR)
		netName  = nibble.NetworkName()
		wgName   = nibble.WiregardName()
	)

	dir, err := ioutil.TempDir("", netName)
	require.NoError(t, err)

	storage := filepath.Join(dir, netName)

	defer func() {
		_ = deleteNetResource(&resource, network.AllocationNR)
		_ = os.RemoveAll(dir)
	}()

	err = createNetworkResource(network.NetID, &resource, network.AllocationNR)
	require.NoError(t, err)

	key, err := configureWG(storage, &resource, network.AllocationNR)
	assert.NoError(t, err)

	netns, err := namespace.GetByName(netName)
	require.NoError(t, err)

	// verifty the configfuration of the wg interface
	// for this we need to swtich into the network namespace created for
	// this network resource
	var handler = func(_ ns.NetNS) error {
		wg, err := wireguard.GetByName(wgName)
		require.NoError(t, err)

		device, err := wg.Device()
		require.NoError(t, err)

		assert.Equal(t, wgName, device.Name)
		assert.Equal(t, key, device.PrivateKey)
		assert.Equal(t, key.PublicKey(), device.PublicKey)

		for i, peer := range device.Peers {
			// asserts endpoint
			assert.Equal(t, endpoint(resource.Connected[i]), peer.Endpoint.String())

			// asserts allowedIPs
			a, b, err := nibble.ToV4()
			require.NoError(t, err)
			expected := []string{
				fmt.Sprintf("fe80::%s/128", nibble.Hex()),
				fmt.Sprintf("172.16.%d.%d/32", a, b),
			}
			actual := make([]string, len(peer.AllowedIPs))
			for y, ip := range peer.AllowedIPs {
				actual[y] = ip.String()
			}
			assert.Equal(t, expected, actual)
		}
		return nil
	}
	err = netns.Do(handler)
	require.NoError(t, err)

}

func mustParseCIDR(cidr string) net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return *ipnet
}
