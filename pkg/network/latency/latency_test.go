package latency_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/network/latency"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
)

func TestLatency(t *testing.T) {
	l, err := latency.Latency("explorer.grid.tf:80")
	require.NoError(t, err)
	t.Log(l)
}

func TestLatencySorter(t *testing.T) {
	ls := latency.NewSorter([]string{
		"explorer.grid.tf:80",
		"google.com:80",
	}, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ls.Run(ctx)
	for _, r := range results {
		fmt.Printf("%s %v\n", r.Endpoint, r.Latency)
	}
	assert.Equal(t, 2, len(results))
}

func TestLatencySorterIPV4Only(t *testing.T) {
	ls := latency.NewSorter([]string{
		"tcp://[2a00:1450:400e:806::200e]:443",
		"tcp://172.217.17.78:443",
	}, 1, latency.IPV4Only)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ls.Run(ctx)
	assert.Equal(t, 1, len(results))
}

func TestYggPeering(t *testing.T) {
	pl, err := yggdrasil.FetchPeerList()
	require.NoError(t, err)

	peersUp := pl.Ups()
	endpoints := make([]string, len(peersUp))
	for i, p := range peersUp {
		endpoints[i] = p.Endpoint
	}

	ls := latency.NewSorter(endpoints, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ls.Run(ctx)
	for _, r := range results {
		fmt.Printf("%30s %v\n", r.Endpoint, r.Latency)
	}
}

func TestIPV4Only(t *testing.T) {
	for _, tc := range []struct {
		ip   net.IP
		ipv4 bool
	}{
		{
			ip:   net.ParseIP("2a00:1450:400e:80a::2004"),
			ipv4: false,
		},
		{
			ip:   net.ParseIP("82.118.227.155"),
			ipv4: true,
		},
	} {
		t.Run(tc.ip.String(), func(t *testing.T) {
			assert.Equal(t, tc.ipv4, latency.IPV4Only(tc.ip))
		})
	}
}

func TestExcludePrefix(t *testing.T) {
	for _, tc := range []struct {
		ip     net.IP
		prefix []byte
		expect bool
	}{
		{
			ip:     net.ParseIP("2a00:1450:400e:80a::2004"),
			prefix: net.ParseIP("2a02:1802:5e:0::"),
			expect: true,
		},
		{
			ip:     net.ParseIP("2a02:1802:5e:0:18d2:e2ff:fe44:17d2"),
			prefix: net.ParseIP("2a02:1802:5e:0::"),
			expect: false,
		},
	} {
		t.Run(tc.ip.String(), func(t *testing.T) {
			assert.Equal(t, tc.expect, latency.ExcludePrefix(tc.prefix[:8])(tc.ip))
		})
	}

}
