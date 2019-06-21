package ip

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNibble(t *testing.T) {
	for _, tc := range []struct {
		prefix  net.IPNet
		allocNr int8
		wg      string
		bridge  string
		veth    string
		network string
	}{
		{
			prefix:  mustParseCIDR("2a02:1802:5e:ff02::/64"),
			allocNr: 0,
			wg:      "wg-ff02-0",
			bridge:  "br-ff02-0",
			veth:    "veth-ff02-0",
			network: "net-ff02-0",
		},
		{
			prefix:  mustParseCIDR("2a02:1802:5e:ff02::/64"),
			allocNr: 2,
			wg:      "wg-ff02-2",
			bridge:  "br-ff02-2",
			veth:    "veth-ff02-2",
			network: "net-ff02-2",
		},
		{
			prefix:  mustParseCIDR("2a02:1802:5e:ff02::/48"),
			allocNr: 2,
			wg:      "wg-ff02-2",
			bridge:  "br-ff02-2",
			veth:    "veth-ff02-2",
			network: "net-ff02-2",
		},
		{
			prefix:  mustParseCIDR("2a02:1802:5e:ff02::/40"),
			allocNr: 0,
			wg:      "wg-005eff02-0",
			bridge:  "br-005eff02-0",
			veth:    "veth-005eff02-0",
			network: "net-005eff02-0",
		},
	} {
		name := fmt.Sprintf("%s-%d", tc.prefix.String(), tc.allocNr)
		t.Run(name, func(t *testing.T) {
			nibble := NewNibble(&tc.prefix, tc.allocNr)
			assert.Equal(t, tc.wg, nibble.WiregardName())
			assert.Equal(t, tc.bridge, nibble.BridgeName())
			assert.Equal(t, tc.veth, nibble.VethName())
			assert.Equal(t, tc.network, nibble.NetworkName())
		})
	}
}

func mustParseCIDR(cidr string) net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return *ipnet
}
