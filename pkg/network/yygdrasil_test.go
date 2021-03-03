package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"gotest.tools/assert"
)

func TestGenerateConfig(t *testing.T) {
	kp, err := identity.FromSeed([]byte("00000000000000000000000000000000"))
	require.NoError(t, err)

	cfg := yggdrasil.GenerateConfig(kp.PrivateKey)

	assert.Equal(t, "30303030303030303030303030303030303030303030303030303030303030301ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", cfg.SigningPrivateKey)
	assert.Equal(t, "1ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", cfg.SigningPublicKey)
	assert.Equal(t, "98b6d128682e280b74b324ca82a6bae6e8a3f7174e0605bfd52eb9948fad8944", cfg.EncryptionPrivateKey)
	assert.Equal(t, "167315f5a03796214692d3a74b0a48d630f7fa5e5257730da0415dc7af7ab260", cfg.EncryptionPublicKey)
	assert.Equal(t, "2ru5Pc", cfg.NodeInfo["name"])
}

func TestAddresses(t *testing.T) {
	kp, err := identity.FromSeed([]byte("00000000000000000000000000000000"))
	require.NoError(t, err)

	cfg := yggdrasil.GenerateConfig(kp.PrivateKey)
	s := NewYggServer(nil, &cfg)

	ip, err := s.Address()
	require.NoError(t, err)

	subnet, err := s.Subnet()
	require.NoError(t, err)

	gw, err := s.Gateway()
	require.NoError(t, err)

	assert.Equal(t, "201:8331:6ca1:39c7:f468:fab5:2bb9:daf4", ip.String())
	assert.Equal(t, "301:8331:6ca1:39c7::/64", subnet.String())
	assert.Equal(t, "301:8331:6ca1:39c7::1/64", gw.String())
}

func TestSubnetFor(t *testing.T) {
	tests := []struct {
		name   string
		prefix net.IP
		ip     net.IP
		b      []byte
	}{
		{
			name:   "0000",
			prefix: net.ParseIP("301:8d90:378f:a93::"),
			ip:     net.ParseIP("301:8d90:378f:a93:c600:1d5b:2ac3:df31"),
			b:      []byte("0000"),
		},
		{
			name:   "bbbbb",
			prefix: net.ParseIP("301:8d90:378f:a93::"),
			ip:     net.ParseIP("301:8d90:378f:a93:9b73:d9aa:7cec:73be"),
			b:      []byte("bbbb"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := subnetFor(tt.prefix, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.ip.String(), ip.String())
		})
	}
}
