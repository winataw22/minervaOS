package provision

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

func TestZDBProvisionMappingExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestZDBCache
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithZDBMapping(ctx, &cache)

	zdb := ZDB{
		Size:     280682,
		Mode:     pkg.ZDBModeSeq,
		Password: "pa$$w0rd",
		DiskType: pkg.SSDDevice,
		Public:   true,
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ZDBReservation,
		Data: MustMarshal(t, zdb),
	}

	cache.On("Get", reservation.ID).Return("container-id", true)

	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect", "zdb", pkg.ContainerID("container-id")).
		Return(pkg.Container{
			Name: "container-id",
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
		}, nil)

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Addrs",
		"zdb0", "net-ns").Return([]net.IP{net.ParseIP("2001:cdba::3257:9652")}, nil)

	result, err := zdbProvisionImpl(ctx, &reservation)

	require.NoError(err)
	require.Equal(reservation.ID, result.Namespace)
	require.Equal("2001:cdba::3257:9652", result.IP)
	require.EqualValues(zdbPort, result.Port)
}

func TestZDBProvisionNoMappingContainerExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestZDBCache
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithZDBMapping(ctx, &cache)

	zdb := ZDB{
		Size:     10,
		Mode:     pkg.ZDBModeSeq,
		Password: "pa$$w0rd",
		DiskType: pkg.SSDDevice,
		Public:   true,
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ZDBReservation,
		Data: MustMarshal(t, zdb),
	}

	// return false on cache force creation
	cache.On("Get", reservation.ID).Return("", false)

	// it's followed by allocation request to see if there is already a
	// zdb instance running that has enough space for the ns

	client.On("Request", "storage", zbus.ObjectID{Name: "storage", Version: "0.0.1"},
		"Allocate",
		zdb.DiskType, zdb.Size*gigabyte, zdb.Mode,
	).Return("container-id", "/path/to/volume", nil)

	// container exists, so inspect still should return No error
	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect", "zdb", pkg.ContainerID("container-id")).
		Return(pkg.Container{
			Name: "container-id",
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
		}, nil)

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Addrs",
		"zdb0", "net-ns").Return([]net.IP{net.ParseIP("2001:cdba::3257:9652")}, nil)

	_, err := zdbProvisionImpl(ctx, &reservation)
	// unfortunately the provision will try (as last step) to dial
	// the unix socket of the server, and then actually configure
	// the namespace and set the limits, password, etc...
	// for now we just handle the connection error and assume
	// it succeeded.
	// TODO: start a mock server on this address
	require.Error(err)
	require.True(strings.Contains(err.Error(), "failed to connect to 0-db at unix:///var/run/zdb_container-id/zdb.sock"))
}

func TestZDBProvisionNoMappingContainerDoesNotExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestZDBCache
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithZDBMapping(ctx, &cache)

	zdb := ZDB{
		Size:     10,
		Mode:     pkg.ZDBModeSeq,
		Password: "pa$$w0rd",
		DiskType: pkg.SSDDevice,
		Public:   true,
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ZDBReservation,
		Data: MustMarshal(t, zdb),
	}

	// return false on cache force creation
	cache.On("Get", reservation.ID).Return("", false)

	// it's followed by allocation request to see if there is already a
	// zdb instance running that has enough space for the ns

	client.On("Request", "storage", zbus.ObjectID{Name: "storage", Version: "0.0.1"},
		"Allocate",
		zdb.DiskType, zdb.Size*gigabyte, zdb.Mode,
	).Return("container-id", "/path/to/volume", nil)

	// container does NOT exists, so inspect still should return an error
	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect", "zdb", pkg.ContainerID("container-id")).
		Return(pkg.Container{}, fmt.Errorf("not found"))

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Addrs",
		"zdb0", "net-ns").Return([]net.IP{net.ParseIP("2001:cdba::3257:9652")}, nil)

	client.On("Request", "flist", zbus.ObjectID{Name: "flist", Version: "0.0.1"},
		"Mount",
		"https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-development.flist",
		"", pkg.MountOptions{
			Limit:    10,
			ReadOnly: false,
		}).Return("/path/to/volume", nil)

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"ZDBPrepare").Return("net-ns", nil)

	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Run",
		"zdb",
		pkg.Container{
			Name:    "container-id",
			RootFS:  "/path/to/volume",
			Network: pkg.NetworkInfo{Namespace: "net-ns"},
			Mounts: []pkg.MountInfo{
				pkg.MountInfo{
					Source: "/path/to/volume",
					Target: "/data",
				},
				pkg.MountInfo{
					Source: "/var/run/zdb_container-id",
					Target: "/socket",
				},
			},
			Entrypoint:  "/bin/zdb --data /data --index /data --mode seq  --listen :: --port 9900 --socket /socket/zdb.sock --dualnet",
			Interactive: false,
		},
	).Return(pkg.ContainerID("container-id"), nil)

	cache.On("Set", reservation.ID, "container-id")

	_, err := zdbProvisionImpl(ctx, &reservation)

	// unfortunately the provision will try (as last step) to dial
	// the unix socket of the server, and then actually configure
	// the namespace and set the limits, password, etc...
	// for now we just handle the connection error and assume
	// it succeeded.
	// TODO: start a mock server on this address
	require.Error(err)
	require.True(strings.Contains(err.Error(), "/var/run/zdb_container-id"))
}

func Test_findDataVolume(t *testing.T) {
	type args struct {
		mounts []pkg.MountInfo
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "exists",
			args: args{
				mounts: []pkg.MountInfo{
					{
						Target: "/data",
						Source: "/mnt/sda/subovl1",
					},
					{
						Target: "/etc/resolv.conf",
						Source: "/etc/resolv.conf",
					},
				},
			},
			want:    "subovl1",
			wantErr: false,
		},
		{
			name: "non-exists",
			args: args{
				mounts: []pkg.MountInfo{
					{
						Target: "/etc/resolv.conf",
						Source: "/etc/resolv.conf",
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "empty",
			args: args{
				mounts: []pkg.MountInfo{},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findDataVolume(tt.args.mounts)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
