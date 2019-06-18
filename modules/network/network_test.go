package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/threefoldtech/zosv2/modules/network/bridge"
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

func TestCreateBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := bridge.New(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(br)
	}()

	bridges, err := bridge.List()
	require.NoError(t, err)

	found := false
	for _, link := range bridges {
		fmt.Println(link.Attrs().Name)
		if link.Type() == "bridge" && link.Attrs().Name == ifName {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestAttachBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := bridge.New(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(br)
	}()

	err = bridge.AttachNic(&netlink.Device{LinkAttrs: netlink.LinkAttrs{Name: "dummy0"}}, br)
	assert.NoError(t, err)
}

var networks = []modules.Network{
	{
		NetID: "net1",
		Resources: []modules.NetResource{
			{
				NodeID: modules.NodeID("node1"),
				Prefix: MustParseCIDR("2a02:1802:5e:f002::/64"),
				Connected: []modules.Connected{
					{
						Type:   modules.ConnTypeWireguard,
						Prefix: MustParseCIDR("2a02:1802:5e:cc02::/64"),
						Connection: modules.Wireguard{
							NICName:  "cc02",
							Peer:     net.ParseIP("2001::1"),
							PeerPort: 51820,
							Key:      "4w4woC+AuDUAaRipT49M8SmTkzERps3xA5i0BW4hPiw=",
							// LinkLocal: net.
						},
					},
				},
			},
		},
		// Exit: modules.ExitPoint{

		// },
	},
}

func TestCreateNetwork(t *testing.T) {
	network := networks[0]
	netName := string(network.NetID)

	defer func() {
		_ = deleteNetwork(netName)
	}()

	err := createNetwork(netName)
	require.NoError(t, err)

	assert.True(t, bridge.Exists(bridgeName(netName)))
	assert.True(t, namespace.Exists(netnsName(netName)))

	nsCtx := namespace.NSContext{}
	err = nsCtx.Enter(netnsName(netName))
	require.NoError(t, err)

	_, err = wireguard.GetByName(wgName(netName))
	assert.NoError(t, err)

	err = nsCtx.Exit()
	require.NoError(t, err)

	// createNetwork should be idempotent,
	// so calling it a second time should be a no-op
	err = createNetwork(netName)
	require.NoError(t, err)
}

func TestConfigureWG(t *testing.T) {
	var (
		network = networks[0]
		netName = string(network.NetID)
	)

	dir, err := ioutil.TempDir("", netName)
	require.NoError(t, err)

	storage := filepath.Join(dir, netName)

	defer func() {
		_ = deleteNetwork(netName)
		_ = os.RemoveAll(dir)
	}()

	err = createNetwork(netName)
	require.NoError(t, err)

	err = configureWG(storage, netName, network.Resources[0])
	assert.NoError(t, err)
}
