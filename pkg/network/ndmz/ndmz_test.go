package ndmz

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpv6(t *testing.T) {

	tt := []struct {
		ipv4 net.IP
		ipv6 net.IP
	}{
		{
			ipv4: net.ParseIP("100.127.0.3"),
			ipv6: net.ParseIP("fd00::0000:0003"),
		},
		{
			ipv4: net.ParseIP("100.127.1.1"),
			ipv6: net.ParseIP("fd00::101"),
		},
		{
			ipv4: net.ParseIP("100.127.255.254"),
			ipv6: net.ParseIP("fd00::fffe"),
		},
	}
	for _, tc := range tt {
		ipv6 := convertIpv4ToIpv6(tc.ipv4)
		assert.Equal(t, tc.ipv6, ipv6)
	}

}
