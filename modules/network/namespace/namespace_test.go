package namespace

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containernetworking/plugins/pkg/ns"

	"github.com/vishvananda/netlink"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNetNS(t *testing.T) {
	name := "testns"
	_, err := Create(name)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(netNSPath, name))
	assert.NoError(t, err)

	out, err := exec.Command("ip", "netns").CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(out), name))

	err = Delete(name)
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

	_, err = Create("testns")
	require.NoError(t, err)
	defer Delete("testns")

	err = SetLink(link, "testns")
	assert.NoError(t, err)
}

func printIfaces(ifaces []netlink.Link) {
	for _, iface := range ifaces {
		fmt.Println(iface.Attrs().Name)
	}
}
func TestNamespace(t *testing.T) {
	ifaces, err := netlink.LinkList()
	require.NoError(t, err)
	ifacesNr := len(ifaces)
	assert.True(t, ifacesNr > 0)

	nsName := "testns"
	netns, err := Create(nsName)
	require.NoError(t, err)
	defer Delete(nsName)

	err = netns.Do(func(_ ns.NetNS) error {
		ifaces, err = netlink.LinkList()
		require.NoError(t, err)
		assert.True(t, len(ifaces) == 1)

		err = netlink.LinkAdd(&netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: "dummy1",
			},
		})
		require.NoError(t, err)

		ifaces, err = netlink.LinkList()
		require.NoError(t, err)
		assert.True(t, len(ifaces) == 2)
		return nil
	})
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

func TestAddRoute(t *testing.T) {
	t.SkipNow()
	nsName := "testns"
	_, err := Create(nsName)
	require.NoError(t, err)
	defer Delete(nsName)

	err = RouteAdd(nsName, &netlink.Route{
		Src: net.ParseIP("172.21.0.10"),
		Gw:  net.ParseIP("172.21.0.1"),
	})
	require.NoError(t, err)
}
