package zdb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestReadHeader(t *testing.T) {
	f, err := os.Open("./test_data/zdb-namespace.v2")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadHeaderV2(f)
	require.NoError(t, err)

	assert.Equal(t, uint32(1), h.Version)
	assert.Equal(t, "test", h.Name)
	assert.Equal(t, "password", h.Password)
	assert.Equal(t, gridtypes.Unit(10240), h.MaxSize) // 1G
}
