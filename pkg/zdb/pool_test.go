package zdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestNamespaces(t *testing.T) {
	pool := Index{
		root: "./test_data/pool_layout",
	}

	ns, err := pool.Namespaces()
	require.NoError(t, err)

	require.Equal(t, 2, len(ns))
	assert.Equal(t, "test", ns[0].Name)
	assert.Equal(t, gridtypes.Unit(10240), ns[0].Size)

	assert.Equal(t, "test2", ns[1].Name)
	assert.Equal(t, gridtypes.Unit(0), ns[1].Size)
}

func TestReserved(t *testing.T) {
	pool := Index{
		root: "./test_data/pool_layout",
	}

	size, err := pool.Reserved()
	require.NoError(t, err)
	assert.Equal(t, uint64(10240), size)
}

func TestExists(t *testing.T) {
	pool := Index{
		root: "./test_data/pool_layout",
	}

	assert.True(t, pool.Exists("test"))
	assert.False(t, pool.Exists("foo"))
}

func TestIndexMode(t *testing.T) {
	pool := Index{
		root: "./test_data/pool_layout",
	}

	mode, err := pool.IndexMode("test")
	require.NoError(t, err)
	assert.Equal(t, IndexModeKeyValue, mode)
}
