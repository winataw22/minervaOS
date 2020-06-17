package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSelf(t *testing.T) {
	t.SkipNow()
	c := NewYggdrasil("tcp://localhost:8999")
	defer c.Close()

	err := c.Connect()
	require.NoError(t, err)

	info, err := c.GetSelf()
	require.NoError(t, err)
	assert.NotEmpty(t, info.BuildName)
	assert.NotEmpty(t, info.BoxPubKey)
	assert.NotEmpty(t, info.IPv6Addr)
	assert.NotEmpty(t, info.Coords)
	assert.NotEmpty(t, info.Subnet)
}

func TestGetPeers(t *testing.T) {
	t.SkipNow()
	c := NewYggdrasil("tcp://localhost:8999")
	defer c.Close()

	err := c.Connect()
	require.NoError(t, err)

	peers, err := c.GetPeers()
	require.NoError(t, err)
	assert.True(t, len(peers) >= 1)
}

func TestAddPeer(t *testing.T) {
	t.SkipNow()
	c := NewYggdrasil("tcp://localhost:8999")
	defer c.Close()

	err := c.Connect()
	require.NoError(t, err)

	added, err := c.AddPeer("tcp://[2001:8d8:1800:8224::1]:6121")
	require.NoError(t, err)
	assert.True(t, len(added) >= 1)
}
