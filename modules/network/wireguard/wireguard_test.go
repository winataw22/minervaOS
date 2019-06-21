package wireguard

import (
	"testing"

	"github.com/vishvananda/netlink"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPeer(t *testing.T) {
	endpoint := "37.187.124.71:51820"
	publicKey := "mR5fBXohKe2MZ6v+GLwlKwrvkFxo1VvV3bPNHDBhOAI="
	allowedIps := []string{"172.21.0.0/24", "fe80::f002/64"}
	peer, err := newPeer(publicKey, endpoint, allowedIps)
	require.NoError(t, err)

	require.Equal(t, endpoint, peer.Endpoint.String())
	require.Equal(t, publicKey, peer.PublicKey.String())
	tmp := make([]string, len(peer.AllowedIPs))
	for i, ip := range peer.AllowedIPs {
		tmp[i] = ip.String()
	}
	require.Equal(t, allowedIps, tmp)
}

func TestConfigure(t *testing.T) {
	wg, err := New("test")
	require.NoError(t, err)

	defer func() {
		_ = netlink.LinkDel(wg)
	}()

	privateKey := "4DwTbGRWECH8oqcTXdoWXGOaWWC952QKbFE1fMzBNmA="
	publicKey := "kDd5mB6L4gkd3U5W287JeQu7urFzBYH51JQZUrJd8Hg="
	endpoint := "37.187.124.71:51820"
	peerPublicKey := "mR5fBXohKe2MZ6v+GLwlKwrvkFxo1VvV3bPNHDBhOAI="
	allowedIps := []string{"172.21.0.0/24", "192.168.1.10/32", "fe80::f002/128"}

	err = wg.Configure(privateKey, []Peer{
		{
			PublicKey:  peerPublicKey,
			AllowedIPs: allowedIps,
			Endpoint:   endpoint,
		},
	})
	require.NoError(t, err)

	device, err := wg.Device()
	require.NoError(t, err)

	assert.Equal(t, privateKey, device.PrivateKey.String())
	assert.Equal(t, publicKey, device.PrivateKey.PublicKey().String())
	assert.Equal(t, publicKey, device.PrivateKey.PublicKey().String())

	for _, peer := range device.Peers {
		assert.Equal(t, endpoint, peer.Endpoint.String())

		actual := make([]string, len(peer.AllowedIPs))
		for y, ip := range peer.AllowedIPs {
			actual[y] = ip.String()
		}
		assert.Equal(t, allowedIps, actual)
	}
}
