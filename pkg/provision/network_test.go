package provision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func TestNetworkProvision(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	const module = "network"
	version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	network := pkg.Network{
		Name: "net1",
		// we set netid here ourselves although it's set by the provision
		// method just to make sure that assertion pass.
		NetID:   networkID("user", "net1"),
		IPRange: types.MustParseIPNet("192.168.1.0/24"),
		NetResources: []pkg.NetResource{
			{
				NodeID: "node-1",
				Subnet: types.MustParseIPNet("192.168.1.0/24"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: NetworkReservation,
		Data: MustMarshal(t, network),
	}

	client.On("Request", module, version, "CreateNR", network).
		Return("ns", nil)

	err := networkProvisionImpl(ctx, &reservation)
	require.NoError(err)
}

func TestNetworkDecommission(t *testing.T) {
	require := require.New(t)

	var client TestClient
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)

	const module = "network"
	version := zbus.ObjectID{Name: "network", Version: "0.0.1"}

	network := pkg.Network{
		Name: "net1",
		// we set netid here ourselves although it's set by the provision
		// method just to make sure that assertion pass.
		NetID:   networkID("user", "net1"),
		IPRange: types.MustParseIPNet("192.168.1.0/24"),
		NetResources: []pkg.NetResource{
			{
				NodeID: "node-1",
				Subnet: types.MustParseIPNet("192.168.1.0/24"),
			},
		},
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: NetworkReservation,
		Data: MustMarshal(t, network),
	}

	client.On("Request", module, version, "DeleteNR", network).
		Return(nil)

	err := networkDecommission(ctx, &reservation)
	require.NoError(err)
}

func Test_networkID(t *testing.T) {
	type args struct {
		userID string
		name   string
	}
	tests := []struct {
		name string
		args args
		want pkg.NetID
	}{
		{
			name: "net-1",
			args: args{
				userID: "user1",
				name:   "net-1",
			},
			want: pkg.NetID("EJyFexd14LVGi"),
		},
		{
			name: "net-2",
			args: args{
				userID: "user1",
				name:   "net-2",
			},
			want: pkg.NetID("4ftuPjY3wuvho"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := networkID(tt.args.userID, tt.args.name)
			assert.Equal(t, tt.want, got)
		})
	}
}
