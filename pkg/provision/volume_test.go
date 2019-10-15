package provision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

func TestVolumeProvisionExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	const module = "storage"
	version := zbus.ObjectID{Name: "storage", Version: "0.0.1"}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: VolumeReservation,
		Data: MustMarshal(t, Volume{}),
	}

	client.On("Request", module, version, "Path", reservation.ID).
		Return("/some/path", nil)

	result, err := volumeProvisionImpl(ctx, &reservation)
	require.NoError(err)
	require.EqualValues(VolumeResult{"reservation-id"}, result)
}

func TestVolumeProvisionNew(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	const module = "storage"
	version := zbus.ObjectID{Name: "storage", Version: "0.0.1"}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: VolumeReservation,
		Data: MustMarshal(t, Volume{
			Size: 10,
			Type: SSDDiskType,
		}),
	}

	// force creation by returning an error
	client.On("Request", module, version, "Path", reservation.ID).
		Return("", zbus.RemoteError{"does not exist"})

	client.On("Request", module, version, "CreateFilesystem", reservation.ID, 10*gigabyte, pkg.DeviceType(SSDDiskType)).
		Return("/some/path", nil)

	result, err := volumeProvisionImpl(ctx, &reservation)
	require.NoError(err)
	require.EqualValues(VolumeResult{"reservation-id"}, result)
}

func TestVolumeDecomission(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	const module = "storage"
	version := zbus.ObjectID{Name: "storage", Version: "0.0.1"}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: VolumeReservation,
	}

	// force decomission by returning a nil error
	client.On("Request", module, version, "ReleaseFilesystem", reservation.ID).
		Return(nil)

	err := volumeDecommission(ctx, &reservation)
	require.NoError(err)
}
