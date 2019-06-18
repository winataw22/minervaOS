package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

func TestCreateBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(br)
	}()

	bridges, err := List()
	require.NoError(t, err)

	found := false
	for _, link := range bridges {
		if link.Type() == "bridge" && link.Attrs().Name == ifName {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestAttachBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(br)
	}()

	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "dummy0"}}
	err = netlink.LinkAdd(dummy)
	require.NoError(t, err)

	err = AttachNic(dummy, br)
	assert.NoError(t, err)
}
