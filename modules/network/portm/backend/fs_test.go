package backend

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReserve(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	defer func() {
		os.RemoveAll(dir)
	}()

	ns1 := "ns"
	ns2 := "ns2"

	store, err := NewFSStore(dir)
	require.NoError(t, err)

	reserved, err := store.Reserve(ns1, 1)
	require.NoError(t, err)
	assert.True(t, reserved)

	reserved, err = store.Reserve(ns1, 1)
	require.NoError(t, err)
	assert.False(t, reserved, "should not be able to reserve the same port twice")

	err = store.Release(ns1, 1)
	require.NoError(t, err)

	reserved, err = store.Reserve(ns1, 1)
	require.NoError(t, err)
	assert.True(t, reserved, "should be able to reserve a released port")

	reserved, err = store.Reserve(ns2, 1)
	require.NoError(t, err)
	assert.True(t, reserved, "should be able to reserve same port in difference namespace")
}
