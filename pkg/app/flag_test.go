package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteFlag(t *testing.T) {
	key := LimitedCache
	err := DeleteFlag(key)
	require.NoError(t, err)
}

func TestDeleteBadFlag(t *testing.T) {
	key := "/"
	err := DeleteFlag(key)
	require.Error(t, err)
}
