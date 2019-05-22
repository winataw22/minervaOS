package network

import (
	"fmt"
	"testing"

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
	bridge, err := CreateBridge(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(bridge)
	}()

	links, err := interfaces()
	require.NoError(t, err)
	found := false

	for _, link := range filterBridge(links) {
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
	bridge, err := CreateBridge(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(bridge)
	}()

	err = BridgeAttachNic(&netlink.Device{netlink.LinkAttrs{Name: "dummy0"}}, bridge)
	assert.NoError(t, err)
}
