package zdbpool

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadHeader(t *testing.T) {
	f, err := os.Open("./test_data/zdb-namespace")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadHeader(f)
	require.NoError(t, err)

	assert.Equal(t, "test", h.Name)
	assert.Equal(t, "", h.Password)
	assert.Equal(t, uint64(1025), h.MaxSize)
}

func TestReadHeaderExtended(t *testing.T) {
	f, err := os.Open("./test_data/zdb-namespace.v1")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadHeader(f)
	require.NoError(t, err)

	assert.Equal(t, uint32(1), h.Version)
	assert.Equal(t, "test", h.Name)
	assert.Equal(t, "password", h.Password)
	assert.Equal(t, uint64(1073741824), h.MaxSize) // 1G
}

func TestWriteHeader(t *testing.T) {
	var buffer bytes.Buffer
	head := Header{
		Name:     "created",
		Password: "password",
		MaxSize:  10 * 1024 * 1024 * 1024, // 10G
	}

	err := WriteHeader(&buffer, head)
	require.NoError(t, err)

	loaded, err := ReadHeader(&buffer)
	require.NoError(t, err)

	assert.Equal(t, head.Name, loaded.Name)
	assert.Equal(t, head.Password, loaded.Password)
	assert.Equal(t, head.MaxSize, loaded.MaxSize)
	assert.Equal(t, uint32(1), loaded.Version)
}
