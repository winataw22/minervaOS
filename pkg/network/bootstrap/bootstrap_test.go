package bootstrap

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

var _testIfaceCfgs = []IfaceConfig{
	{
		Name: "eth0",
		Addrs4: []netlink.Addr{
			mustParseAddr("172.20.0.96/24"),
		},
		Addrs6: []netlink.Addr{
			mustParseAddr("2001:b:a:0:5645:46ff:fef6:261/64"),
			mustParseAddr("fe80::5645:46ff:fef6:261/64"),
		},
		DefaultGW: net.ParseIP("172.20.0.1"),
	},
	{
		Name: "eth1",
		Addrs4: []netlink.Addr{
			mustParseAddr("172.20.0.97/24"),
		},
		Addrs6: []netlink.Addr{
			mustParseAddr("2001:b:a:0:5645:46ff:fef6:262/64"),
			mustParseAddr("fe80::5645:46ff:fef6:262/64"),
		},
		DefaultGW: net.ParseIP("172.20.0.1"),
	},
}

func TestSelectZOS(t *testing.T) {

	ifaceName, err := SelectZOS(_testIfaceCfgs)
	require.NoError(t, err)

	assert.Equal(t, "eth0", ifaceName)
}

func TestAddrSet(t *testing.T) {
	s := newAddrSet()
	s.Add(mustParseAddr("192.168.0.1/24"))
	s.Add(mustParseAddr("192.168.0.1/24"))
	s.Add(mustParseAddr("192.168.1.1/24"))

	assert.Equal(t, 2, s.Len())

	s.AddSlice([]netlink.Addr{
		mustParseAddr("192.168.0.1/24"),
		mustParseAddr("192.168.0.10/24"),
	})
	assert.Equal(t, 3, s.Len())
	assert.Equal(t, []netlink.Addr{
		mustParseAddr("192.168.0.1/24"),
		mustParseAddr("192.168.1.1/24"),
		mustParseAddr("192.168.0.10/24"),
	}, s.ToSlice())
}

func mustParseAddr(s string) netlink.Addr {
	addr, err := netlink.ParseAddr(s)
	if err != nil {
		panic(err)
	}
	return *addr
}
