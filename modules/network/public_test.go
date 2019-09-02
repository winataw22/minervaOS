package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/types"
)

func TestCreatePublicNS(t *testing.T) {
	iface := &types.PubIface{
		Master: "zos0",
		Type:   types.MacVlanIface,
		IPv6: &net.IPNet{
			IP:   net.ParseIP("2a02:1802:5e:ff02::100"),
			Mask: net.CIDRMask(64, 128),
		},
		GW6: net.ParseIP("fe80::1"),
	}

	defer func() {
		pubNS, _ := namespace.GetByName(types.PublicNamespace)
		err := namespace.Delete(pubNS)
		require.NoError(t, err)
	}()

	err := CreatePublicNS(iface)
	require.NoError(t, err)
}
