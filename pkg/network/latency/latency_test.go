package latency_test

import (
	"context"
	"fmt"
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

func TestYggPeering(t *testing.T) {
	pl, err := yggdrasil.FetchPeerList()
	require.NoError(t, err)

	peersUp, err := pl.Ups()
	require.NoError(t, err)
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
