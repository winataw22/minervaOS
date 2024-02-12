package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRedisAddress(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expected    RedisDialParams
		expectedErr error
	}{
		{
			name:    "tcp scheme",
			address: "tcp://localhost:6379",
			expected: RedisDialParams{
				Scheme: "tcp",
				Host:   "localhost:6379",
			},
		},
		{
			name:    "unix scheme",
			address: "unix:///var/run/redis.sock",
			expected: RedisDialParams{
				Scheme: "unix",
				Host:   "/var/run/redis.sock",
			},
		},
		{
			name:    "redis scheme",
			address: "redis://localhost:6379",
			expected: RedisDialParams{
				Scheme: "tcp",
				Host:   "localhost:6379",
			},
		},
		{
			name:        "missing scheme",
			address:     "localhost",
			expectedErr: fmt.Errorf("unknown scheme '' expecting tcp or unix"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params, err := parseRedisAddress(test.address)
			require.Equal(t, test.expectedErr, err)

			if err == nil {
				require.Equal(t, test.expected.Scheme, params.Scheme)
				require.Equal(t, test.expected.Host, params.Host)
			}
		})
	}
}
