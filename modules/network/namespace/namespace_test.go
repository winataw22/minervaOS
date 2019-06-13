package namespace

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNetNS(t *testing.T) {
	name := "testns"
	_, err := CreateNetNS(name)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(netNSPath, name))
	assert.NoError(t, err)

	out, err := exec.Command("ip", "netns").CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(out), name))

	err = DeleteNetNS(name)
	require.NoError(t, err)

	out, err = exec.Command("ip", "netns").CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(out), name))
}

func TestSetLinkNS(t *testing.T) {
	link := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: "dummy0",
		},
	}
	err := netlink.LinkAdd(link)
	require.NoError(t, err)
	defer netlink.LinkDel(link)

	_, err = CreateNetNS("testns")
	require.NoError(t, err)
	defer DeleteNetNS("testns")

	err = SetLinkNS(link, "testns")
	assert.NoError(t, err)
}

func printIfaces(ifaces []netlink.Link) {
	for _, iface := range ifaces {
		fmt.Println(iface.Attrs().Name)
	}
}
func TestNamespace(t *testing.T) {
	ifaces, err := netlink.LinkList()
	ifacesNr := len(ifaces)
	assert.True(t, ifacesNr > 0)

	nsName := "testns"
	_, err = CreateNetNS(nsName)
	require.NoError(t, err)
	defer DeleteNetNS(nsName)

	nsCtx := NSContext{}
	err = nsCtx.Enter(nsName)
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	assert.True(t, len(ifaces) == 1)

	err = netlink.LinkAdd(&netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: "dummy1",
		},
	})
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	assert.True(t, len(ifaces) == 2)

	err = nsCtx.Exit()
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	require.NoError(t, err)

	found := false
	for _, iface := range ifaces {
		if iface.Attrs().Name == "dummy1" {
			found = true
		}
	}
	assert.Equal(t, ifacesNr, len(ifaces))
	assert.False(t, found)
}

func TestNSContext(t *testing.T) {
	nsName := "testns"

	origin, err := netns.Get()
	require.NoError(t, err)

	_, err = CreateNetNS(nsName)
	require.NoError(t, err)
	defer DeleteNetNS(nsName)

	nsCtx := NSContext{}
	err = nsCtx.Enter(nsName)
	require.NoError(t, err)

	current, err := netns.Get()
	require.NoError(t, err)

	assert.True(t, nsCtx.origins.Equal(origin))
	assert.True(t, nsCtx.working.Equal(current))

	err = nsCtx.Exit()
	require.NoError(t, err)

	current, _ = netns.Get()
	assert.True(t, origin.Equal(current))
}

func TestAddRoute(t *testing.T) {
	nsName := "testns"
	_, err := CreateNetNS(nsName)
	require.NoError(t, err)
	defer DeleteNetNS(nsName)

	ns, err := netns.GetFromName(nsName)
	require.NoError(t, err)

	h, err := netlink.NewHandleAt(ns)
	require.NoError(t, err)
	err = h.RouteAdd(&netlink.Route{
		Src: net.ParseIP("172.21.0.10"),
		Gw:  net.ParseIP("172.21.0.1"),
	})
	require.NoError(t, err)
}

// func TestCreateNetNSMultiple(t *testing.T) {
// 	name := "testns"
// 	_, err := CreateNetNS(name)
// 	require.NoError(t, err)

// 	name2 := "testns2"
// 	_, err = CreateNetNS(name2)
// 	require.NoError(t, err)

// 	err = DeleteNetNS(name)
// 	require.NoError(t, err)

// 	err = DeleteNetNS(name2)
// 	require.NoError(t, err)
// }
