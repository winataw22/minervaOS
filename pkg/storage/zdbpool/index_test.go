package zdbpool

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadIndexHeader(t *testing.T) {
	f, err := os.Open("./test_data/zdb-index-00000")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadIndex(f)
	require.NoError(t, err)

	assert.Equal(t, IndexModeKeyValue, h.Mode)
}
