package provision

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules/identity"
)

func TestVerifySignature(t *testing.T) {
	keyPair, err := identity.GenerateKeyPair()
	require.NoError(t, err)

	data, err := json.Marshal(Volume{
		Type: SSDDiskType,
		Size: 20,
	})
	require.NoError(t, err)

	r := Reservation{
		ID:   "reservationID",
		User: keyPair.Identity(),
		Type: ContainerReservation,
		Data: data,
	}

	err = r.Sign(keyPair.PrivateKey)
	require.NoError(t, err)

	err = Verify(r)
	assert.NoError(t, err)

	// corrupt the signature
	validByte := r.Signature[0]
	r.Signature[0] = 'a'
	err = Verify(r)
	assert.Error(t, err)

	// restore signature
	r.Signature[0] = validByte
	// sanity test
	err = Verify(r)
	require.NoError(t, err)

	// change the reservation
	r.User = "attackerID"
	err = Verify(r)
	assert.Error(t, err)
}

func TestHash(t *testing.T) {
	data, err := json.Marshal(Volume{
		Type: SSDDiskType,
		Size: 20,
	})
	require.NoError(t, err)

	r := Reservation{
		ID:   "reservationID",
		User: "userID",
		Type: ContainerReservation,
		Data: data,
	}

	hash, err := Hash(r)
	require.NoError(t, err)
	assert.Equal(t, "14df9cd4862605ff4a3cf26711f416f3d887d8793446a282374ac7609caecfde", fmt.Sprintf("%x", hash))
}
