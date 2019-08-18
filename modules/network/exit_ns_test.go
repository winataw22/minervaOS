package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

func TestCreatePublicNS(t *testing.T) {
	iface := &PubIface{
		Master: "zos0",
		Type:   MacVlanIface,
		IPv6:   mustParseCIDR("2a02:1802:5e:ff02::100/64"),
		GW6:    net.ParseIP("fe80::1"),
	}

	defer func() {
		pubNS, _ := namespace.GetByName("public")
		err := namespace.Delete(pubNS)
		require.NoError(t, err)
	}()

	err := CreatePublicNS(iface)
	require.NoError(t, err)
}
