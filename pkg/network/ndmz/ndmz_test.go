package ndmz

import (
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIPv4Allocate(t *testing.T) {
	var (
		err    error
		origin = ipamPath
	)

	ipamPath, err = ioutil.TempDir("", "")
	require.NoError(t, err)
	defer func() { ipamPath = origin }()

	addr, err := allocateIPv4("network1")
	require.NoError(t, err)

	addr2, err := allocateIPv4("network1")
	require.NoError(t, err)

	assert.Equal(t, addr.String(), addr2.String())

	addr3, err := allocateIPv4("network2")
	require.NoError(t, err)
	assert.NotEqual(t, addr.String(), addr3.String())
}

func TestIPv4AllocateConcurent(t *testing.T) {
	var (
		err    error
		origin = ipamPath
	)

	ipamPath, err = ioutil.TempDir("", "")
	require.NoError(t, err)
	defer func() { ipamPath = origin }()

	wg := sync.WaitGroup{}
	wg.Add(10)

	c := make(chan *net.IPNet)

	for i := 0; i < 10; i++ {
		go func(c chan *net.IPNet, i int) {
			defer wg.Done()
			for y := 0; y < 10; y++ {
				nw := fmt.Sprintf("network%d%d", i, y)
				addr, err := allocateIPv4(nw)
				require.NoError(t, err)
				c <- addr
			}
		}(c, i)
	}

	go func() {
		addrs := map[*net.IPNet]struct{}{}
		for addr := range c {
			_, exists := addrs[addr]
			require.False(t, exists)
			addrs[addr] = struct{}{}
		}
	}()

	wg.Wait()
	close(c)
}
