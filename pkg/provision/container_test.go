package provision

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

func TestContainerProvisionExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestOwnerCache

	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithOwnerCache(ctx, &cache)

	// const module = "network"
	// version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	container := Container{
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Network: Network{
			NetworkID: pkg.NetID("net1"),
			IPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ContainerReservation,
		Data: MustMarshal(t, container),
	}

	// first, the provision will inspect the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect",
		fmt.Sprintf("ns%s", reservation.User),
		pkg.ContainerID(reservation.ID)).
		Return(pkg.Container{Name: reservation.ID}, nil)

	result, err := containerProvisionImpl(ctx, &reservation)
	require.NoError(err)
	require.Equal("reservation-id", result.ID)
}

func TestContainerProvisionNew(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestOwnerCache

	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithOwnerCache(ctx, &cache)

	// const module = "network"
	// version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	container := Container{
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Network: Network{
			NetworkID: pkg.NetID("net1"),
			IPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ContainerReservation,
		Data: MustMarshal(t, container),
	}

	userNs := fmt.Sprintf("ns%s", reservation.User)
	// first, the provision will inspect the container
	// see if it exists
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect",
		userNs,
		pkg.ContainerID(reservation.ID)).
		Return(pkg.Container{}, zbus.RemoteError{"does not exist"})

	netID := networkID(reservation.User, string(container.Network.NetworkID))

	// since it's a new container, a call to join the network is made
	client.On("Request",
		"network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Join",
		netID, reservation.ID, []string{"192.168.1.1"}).
		Return(pkg.Member{
			Namespace: "net-ns",
			IPv4:      net.ParseIP("192.168.1.1"),
		}, nil)

	// then another call to mount the flist
	client.On("Request",
		"flist", zbus.ObjectID{Name: "flist", Version: "0.0.1"},
		"Mount",
		container.FList, container.FlistStorage, pkg.DefaultMountOptions).
		Return("/tmp/root", nil)

	// once root is mounted, a final call to contd to run the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Run",
		userNs, pkg.Container{
			Name:   reservation.ID,
			RootFS: "/tmp/root", //returned by flist.Mount
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
		}).
		Return(pkg.ContainerID(reservation.ID), nil)

	result, err := containerProvisionImpl(ctx, &reservation)
	require.NoError(err)
	require.Equal("reservation-id", result.ID)
	require.Equal("192.168.1.1", result.IPv4)
}

func TestContainerProvisionWithMounts(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestOwnerCache

	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithOwnerCache(ctx, &cache)

	// const module = "network"
	// version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	container := Container{
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Network: Network{
			NetworkID: pkg.NetID("net1"),
			IPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
		Mounts: []Mount{
			{
				VolumeID:   "vol1",
				Mountpoint: "/opt",
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ContainerReservation,
		Data: MustMarshal(t, container),
	}

	//NOTE: since we have moutns, a call is made to get the owner
	//of the volume to validate it's the same owner of the reservation
	cache.On("OwnerOf", "vol1").Return(reservation.User, nil)

	userNs := fmt.Sprintf("ns%s", reservation.User)
	// first, the provision will inspect the container
	// see if it exists
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect",
		userNs,
		pkg.ContainerID(reservation.ID)).
		Return(pkg.Container{}, zbus.RemoteError{"does not exist"})

	netID := networkID(reservation.User, string(container.Network.NetworkID))

	// since it's a new container, a call to join the network is made
	client.On("Request",
		"network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Join",
		netID, reservation.ID, []string{"192.168.1.1"}).
		Return(pkg.Member{
			Namespace: "net-ns",
			IPv4:      net.ParseIP("192.168.1.1"),
		}, nil)

	// then another call to mount the flist
	client.On("Request",
		"storage", zbus.ObjectID{Name: "storage", Version: "0.0.1"},
		"Path",
		"vol1").
		Return("/some/path/to/vol1", nil)

	//for each volume in the list, a call is made to get the mount path
	//of that volume
	client.On("Request",
		"flist", zbus.ObjectID{Name: "flist", Version: "0.0.1"},
		"Mount",
		container.FList, container.FlistStorage, pkg.DefaultMountOptions).
		Return("/tmp/root", nil)

	// once root is mounted, a final call to contd to run the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Run",
		userNs, pkg.Container{
			Name:   reservation.ID,
			RootFS: "/tmp/root", //returned by flist.Mount
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
			Mounts: []pkg.MountInfo{
				{
					Source: "/some/path/to/vol1",
					Target: "/opt",
				},
			},
		}).
		Return(pkg.ContainerID(reservation.ID), nil)

	result, err := containerProvisionImpl(ctx, &reservation)
	require.NoError(err)
	require.Equal("reservation-id", result.ID)
	require.Equal("192.168.1.1", result.IPv4)
}

func TestContainerDecomissionExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestOwnerCache

	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithOwnerCache(ctx, &cache)

	// const module = "network"
	// version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	container := Container{
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Network: Network{
			NetworkID: pkg.NetID("net1"),
			IPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ContainerReservation,
		Data: MustMarshal(t, container),
	}
	rootFS := "/mnt/flist_root"

	// first, the decomission will inspect the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect",
		fmt.Sprintf("ns%s", reservation.User),
		pkg.ContainerID(reservation.ID)).
		Return(pkg.Container{Name: reservation.ID, RootFS: rootFS}, nil)

	// second, the decomission will delete the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Delete",
		fmt.Sprintf("ns%s", reservation.User),
		pkg.ContainerID(reservation.ID)).
		Return(nil)

	// third, unmount the flist of the container
	client.On("Request",
		"flist", zbus.ObjectID{Name: "flist", Version: "0.0.1"},
		"Umount",
		rootFS).
		Return(nil)

	netID := networkID(reservation.User, string(container.Network.NetworkID))
	// fourth, leave the container network namespace
	client.On("Request",
		"network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Leave",
		netID,
		reservation.ID).
		Return(nil)

	err := containerDecommission(ctx, &reservation)
	require.NoError(err)
}

func TestContainerDecomissionContainerGone(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestOwnerCache

	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithOwnerCache(ctx, &cache)

	container := Container{
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Network: Network{
			NetworkID: pkg.NetID("net1"),
			IPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ContainerReservation,
		Data: MustMarshal(t, container),
	}

	// first, the decomission will inspect the container
	client.On("Request",
		"container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect",
		fmt.Sprintf("ns%s", reservation.User),
		pkg.ContainerID(reservation.ID)).
		Return(nil, fmt.Errorf("container not found"))

	netID := networkID(reservation.User, string(container.Network.NetworkID))
	// fourth, leave the container network namespace
	client.On("Request",
		"network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Leave",
		netID,
		reservation.ID).
		Return(nil)

	err := containerDecommission(ctx, &reservation)
	require.NoError(err)
}
