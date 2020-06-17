package yggdrasil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPeerList(t *testing.T) {
	pl, err := FetchPeerList()
	require.NoError(t, err)

	count := 0
	for country := range pl.peers {
		for range pl.peers[country] {
			count++
		}
	}
	p2 := pl.Peers()
	assert.Equal(t, count, len(p2))

	for _, peer := range pl.Ups() {
		assert.True(t, peer.Up)
	}
}

func TestUps(t *testing.T) {
	pl, err := FetchPeerList()
	require.NoError(t, err)

	for _, peer := range pl.Ups() {
		assert.True(t, peer.Up)
	}
}
