package yggdrasil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPeerList(t *testing.T) {
	pl, err := FetchPubYggPeerList()
	require.NoError(t, err)
	peers, err := pl.Ups()
	require.NoError(t, err)
	for _, peer := range peers {
		assert.True(t, peer.Up)
	}
}

func TestUps(t *testing.T) {
	pl, err := FetchPubYggPeerList()
	require.NoError(t, err)

	peers, err := pl.Ups()
	require.NoError(t, err)

	for _, peer := range peers {
		assert.True(t, peer.Up)
	}
}
