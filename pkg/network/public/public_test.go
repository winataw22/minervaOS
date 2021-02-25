package public

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func TestCreatePublicNS(t *testing.T) {
	iface := &pkg.PublicConfig{
		Type: pkg.MacVlanIface,
		IPv6: gridtypes.IPNet{IPNet: net.IPNet{
			IP:   net.ParseIP("2a02:1802:5e:ff02::100"),
			Mask: net.CIDRMask(64, 128),
		}},
		GW6: net.ParseIP("fe80::1"),
	}

	defer func() {
		pubNS, _ := namespace.GetByName(types.PublicNamespace)
		err := namespace.Delete(pubNS)
		require.NoError(t, err)
	}()

	err := setupPublicNS(pkg.StrIdentifier(""), iface)
	require.NoError(t, err)
}
