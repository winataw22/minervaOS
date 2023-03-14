package zos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidBackend(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("1.1.1.1:10")
		err := backend.Valid(true)
		require.NoError(err)

		backend = Backend("1.1.1.1:10")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("1.1.1.1")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("1.1.1.1:port")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("ip:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("1.1.1.1:1000000")
		err = backend.Valid(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1:10")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://1.1.1.1:port")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("http://ip:10")
		err = backend.Valid(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://ip")
		err = backend.Valid(false)
		require.Error(err)
	})
}

func TestAsAddress(t *testing.T) {

	cases := []struct {
		Backend Backend
		Address string
		Err     bool
	}{
		{
			Backend: Backend("http://10.20.30.40"),
			Address: "10.20.30.40:80",
		},
		{
			Backend: Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200"),
			Address: "[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200",
		},
		{
			Backend: Backend("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF:200"),
			Err:     true,
		},
		{
			Backend: Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200"),
			Address: "[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200",
		},
		{
			Backend: Backend("http://10.20.30.40:500"),
			Address: "10.20.30.40:500",
		},
		{
			Backend: Backend("10.20.30.40:500"),
			Address: "10.20.30.40:500",
		},
	}

	for _, c := range cases {
		t.Run(string(c.Backend), func(t *testing.T) {
			addr, err := c.Backend.AsAddress()
			if c.Err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.Address, addr)
			}
		})
	}
}

func TestValidBackendIP6(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err := backend.Valid(true)
		require.NoError(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:1000000")
		err = backend.Valid(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		err = backend.Valid(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		err = backend.Valid(true)
		require.Error(err)
	})
}
