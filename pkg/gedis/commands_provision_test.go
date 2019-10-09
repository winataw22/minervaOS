package gedis

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	types "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/schema"
)

func TestProvisionGet(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := "id"

	args := Args{
		"gwid": id,
	}

	conn.On("Do", "default.workload_manager.workload_get", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{
			"workload_id": id,
			"content":     "raw message",
		}), nil)

	res, err := gedis.Get(id)

	require.NoError(err)
	require.Equal(id, res.ID)
	require.Equal(`"raw message"`, string(res.Data))
	//require.Equal(node.NodeID, "node-1")
	conn.AssertCalled(t, "Close")
}

func TestProvisionPoll(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	node := pkg.StrIdentifier("node-1")
	time := time.Now()

	args := Args{
		"node_id": node.Identity(),
		"epoch":   time.Unix(),
	}

	conn.On("Do", "default.workload_manager.workloads_list", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{
			"workloads": []types.TfgridReservationWorkload1{
				{WorkloadID: "1", Type: types.TfgridReservationWorkload1TypeVolume},
				{WorkloadID: "2", Type: types.TfgridReservationWorkload1TypeContainer},
			},
		}), nil)

	reservations, err := gedis.Poll(node, false, time) //setting false to true will force epoch to 0

	require.NoError(err)
	require.Len(reservations, 2)
	require.Equal(reservations[1].Type, provision.ContainerReservation)
	conn.AssertCalled(t, "Close")

	args = Args{
		"node_id": node.Identity(),
		"epoch":   0,
	}

	conn.On("Do", "default.workload_manager.workloads_list", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{
			"workloads": []types.TfgridReservationWorkload1{
				{WorkloadID: "1", Type: types.TfgridReservationWorkload1TypeVolume},
				{WorkloadID: "2", Type: types.TfgridReservationWorkload1TypeContainer},
			},
		}), nil)

	reservations, err = gedis.Poll(node, true, time)

	require.NoError(err)
	require.Len(reservations, 2)
	require.Equal(reservations[1].Type, provision.ContainerReservation)
	conn.AssertCalled(t, "Close")

}

func TestProvisionFeedback(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := "101"
	result := provision.Result{
		Type:      provision.ContainerReservation,
		ID:        id,
		Created:   time.Now(),
		State:     "ok",
		Data:      json.RawMessage("returned data"),
		Signature: []byte("signature"),
	}

	args := Args{
		"reservation_id": 101,
		"result": types.TfgridReservationResult1{
			Category:   types.TfgridReservationResult1CategoryContainer,
			WorkloadID: 101,
			DataJSON:   string(result.Data),
			Signature:  result.Signature,
			State:      types.TfgridReservationResult1StateOk,
			Epoch:      schema.Date{result.Created},
		},
	}

	conn.On("Do", "default.workload_manager.set_workload_result", mustMarshal(t, args)).
		Return(nil, nil)

	err := gedis.Feedback(id, &result)

	require.NoError(err)
	conn.AssertCalled(t, "Close")
}

func TestProvisionReserve(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := pkg.StrIdentifier("101")
	reservation := provision.Reservation{
		Type:    provision.ContainerReservation,
		ID:      "10",
		Created: time.Now(),
		Data: json.RawMessage(mustMarshal(t, provision.Container{
			FList:      "http://hub.grid.tf/test/test.flist",
			Entrypoint: "/bin/app",
			Network: provision.Network{
				NetwokID: "123",
				IPs:      []net.IP{net.ParseIP("192.168.1.1")},
			},
		})),
		Signature: []byte("signature"),
	}

	args := Args{
		"reservation": types.TfgridReservation1{
			DataReservation: types.TfgridReservationData1{
				Containers: []types.TfgridReservationContainer1{
					{
						Flist:      "http://hub.grid.tf/test/test.flist",
						Entrypoint: "/bin/app",
						NetworkConnection: []types.TfgridReservationNetworkConnection1{
							{NetworkID: "123", Ipaddress: net.ParseIP("192.168.1.1")},
						},
						Volumes: []types.TfgridReservationContainerMount1{},
					},
				},
			},
		},
	}

	conn.On("Do", "default.workload_manager.reservation_register", mock.MatchedBy(func(in []byte) bool {
		EqualJSON(t, mustMarshal(t, args), in)
		return true
	})).
		Return(mustMarshal(t, Args{
			"id": 10,
		}), nil)

	result, err := gedis.Reserve(&reservation, id)

	require.NoError(err)
	require.Equal("10", result)
	conn.AssertCalled(t, "Close")
}
