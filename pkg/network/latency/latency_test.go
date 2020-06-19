package latency

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
)

func TestLatency(t *testing.T) {
	l, err := Latency("explorer.grid.tf:80")
	require.NoError(t, err)
	t.Log(l)
}

func TestLatencySorter(t *testing.T) {
	ls := NewSorter([]string{
		"explorer.grid.tf:80",
		"google.com:80",
	}, 2, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ls.Run(ctx)
	for _, r := range results {
		fmt.Printf("%s %v\n", r.Endpoint, r.Latency)
	}
	assert.Equal(t, len(ls.endpoints), len(results))
}

func TestLatencySorterIPV4Only(t *testing.T) {
	ls := NewSorter([]string{
		"tcp://[2a00:1450:400e:806::200e]:443",
		"tcp://172.217.17.78:443",
	}, 1, true)

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

	ls := NewSorter(endpoints, 2, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ls.Run(ctx)
	for _, r := range results {
		fmt.Printf("%30s %v\n", r.Endpoint, r.Latency)
	}
}

func TestIsIPv4(t *testing.T) {
	for _, tc := range []struct {
		ip   string
		ipv4 bool
	}{
		{
			ip:   "2406:d500:6:beef:21a2:a10c:6aea",
			ipv4: false,
		},
		{
			ip:   "82.118.227.155",
			ipv4: true,
		},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			assert.Equal(t, tc.ipv4, isIPv4(tc.ip))
		})
	}

}
