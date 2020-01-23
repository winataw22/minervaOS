package provision

import (
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/identity"
)

func TestVerifySignature(t *testing.T) {
	keyPair, err := identity.GenerateKeyPair()
	require.NoError(t, err)

	data, err := json.Marshal(Volume{
		Type: SSDDiskType,
		Size: 20,
	})
	require.NoError(t, err)

	r := &Reservation{
		ID:     "reservationID",
		NodeID: "node1",
		User:   keyPair.Identity(),
		Type:   ContainerReservation,
		Data:   data,
	}

	err = r.Sign(keyPair.PrivateKey)
	require.NoError(t, err)

	err = Verify(r)
	assert.NoError(t, err)

	validSignature := make([]byte, len(r.Signature))
	copy(validSignature, r.Signature)

	// corrupt the signature
	_, err = rand.Read(r.Signature)
	require.NoError(t, err)

	err = Verify(r)
	assert.Error(t, err)

	// restore signature
	copy(r.Signature, validSignature)

	// sanity test
	err = Verify(r)
	require.NoError(t, err)

	// change the reservation
	r.User = "attackerID"
	err = Verify(r)
	assert.Error(t, err)
}
