package provision

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/zdb"
)

type zdbTestClient struct {
	mock.Mock
	zdb.Client
}

func (c *zdbTestClient) Connect() error {
	return nil
}

func (c *zdbTestClient) Close() error {
	return nil
}

func (c *zdbTestClient) CreateNamespace(name string) error {
	return c.Called(name).Error(0)
}

func (c *zdbTestClient) NamespaceSetSize(name string, size uint64) error {
	return c.Called(name, size).Error(0)
}

func (c *zdbTestClient) NamespaceSetPassword(name, password string) error {
	return c.Called(name, password).Error(0)
}

func (c *zdbTestClient) NamespaceSetPublic(name string, public bool) error {
	return c.Called(name, public).Error(0)
}

func (c *zdbTestClient) Exist(name string) (bool, error) {
	args := c.Called(name)
	return args.Bool(0), args.Error(1)
}

func TestZDBProvision(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	conf := ZDB{
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
		Data: MustMarshal(t, conf),
	}

	/*
	   Request(string,zbus.ObjectID,string,string,pkg.DeviceType,uint64,pkg.ZDBMode)
	   		0: "storage"
	   		1: zbus.ObjectID{Name:"storage", Version:"0.0.1"}
	   		2: "Allocate"
	   		3: "reservation-id"
	   		4: "SSD"
	   		5: 0x1121a80000000
	   		6: "seq"

	*/
	client.On("Request", "storage", zbus.ObjectID{Name: "storage", Version: "0.0.1"},
		"Allocate", "reservation-id", conf.DiskType, conf.Size*gigabyte, conf.Mode).
		Return(pkg.Allocation{
			VolumeID:   "zdb-container-id",
			VolumePath: "/tmp/volume/path",
		}, nil)

	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect", "zdb", pkg.ContainerID("zdb-container-id")).
		Return(pkg.Container{
			Name: "volume-id",
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
		}, nil)

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Addrs",
		"zdb0", "net-ns").Return([]net.IP{net.ParseIP("2001:cdba::3257:9652")}, nil)

	var zdbClient zdbTestClient
	zdbClient.On("Exist", reservation.ID).Return(false, nil)
	zdbClient.On("CreateNamespace", reservation.ID).Return(nil)
	zdbClient.On("NamespaceSetPassword", reservation.ID, conf.Password).Return(nil)
	zdbClient.On("NamespaceSetPublic", reservation.ID, conf.Public).Return(nil)
	zdbClient.On("NamespaceSetSize", reservation.ID, conf.Size*gigabyte).Return(nil)

	zdbConnection = func(id pkg.ContainerID) zdb.Client {
		return &zdbClient
	}

	result, err := zdbProvisionImpl(ctx, &reservation)

	require.NoError(err)
	require.Equal(reservation.ID, result.Namespace)
	require.Equal("2001:cdba::3257:9652", result.IP)
	require.EqualValues(zdbPort, result.Port)
}

func TestZDBProvisionNoMappingContainerDoesNotExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

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

	// it's followed by allocation request to see if there is already a
	// zdb instance running that has enough space for the ns

	client.On("Request", "storage", zbus.ObjectID{Name: "storage", Version: "0.0.1"},
		"Allocate",
		reservation.ID, zdb.DiskType, zdb.Size*gigabyte, zdb.Mode,
	).Return(pkg.Allocation{
		VolumeID:   "container-id",
		VolumePath: "/path/to/volume",
	}, nil)

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

	// The hw address in this mock call, is based on the VolumeID above. changing the VolumeID
	// will require this HW address to change
	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"ZDBPrepare", []byte{0x7e, 0xcc, 0xc3, 0x81, 0xa2, 0x2e}).Return("net-ns", nil)

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
