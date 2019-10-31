package main

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestHashProof(t *testing.T) {
	// ensure hasProof always generate the same hash
	// for the same content whatever the sorting in the map

	m := map[string]interface{}{}

	b := make([]byte, 4096)
	for i := 0; i < 500; i++ {
		_, err := rand.Read(b)
		require.NoError(t, err)

		k := fmt.Sprintf("%x", b[2048:])
		v := fmt.Sprintf("%x", b[:2048])
		m[k] = v
	}

	h, err := hashProof(m)
	require.NoError(t, err)
	for i := 0; i < 1000; i++ {
		h2, err := hashProof(m)
		require.NoError(t, err)
		assert.Equal(t, h, h2)
	}
}
