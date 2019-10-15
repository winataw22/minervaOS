package provision

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

type TestPollSource struct {
	mock.Mock
}

func (s *TestPollSource) Poll(nodeID pkg.Identifier, all bool, since time.Time) ([]*Reservation, error) {
	returns := s.Called(nodeID, all, since)
	return returns.Get(0).([]*Reservation), returns.Error(1)
}

func TestHTTPReservationSource(t *testing.T) {
	require := require.New(t)
	var store TestPollSource

	nodeID := pkg.StrIdentifier("node-id")
	source := PollSource(&store, nodeID)
	chn := source.Reservations(context.Background())

	store.On("Poll", nodeID, true, mock.Anything).
		Return([]*Reservation{
			&Reservation{ID: "res-1"},
			&Reservation{ID: "res-2"},
		}, ErrPollEOS)

	reservations := []*Reservation{}
	for res := range chn {
		reservations = append(reservations, res)
	}

	require.Len(reservations, 2)
	require.Equal("res-1", reservations[0].ID)
	require.Equal("res-2", reservations[1].ID)
}

func TestHTTPReservationSourceMultiple(t *testing.T) {
	require := require.New(t)
	var store TestPollSource

	nodeID := pkg.StrIdentifier("node-id")
	source := PollSource(&store, nodeID)

	// don't delay the test for long
	source.(*pollSource).maxSleep = 500 * time.Microsecond

	chn := source.Reservations(context.Background())

	store.On("Poll", nodeID, true, mock.Anything).
		Return([]*Reservation{
			&Reservation{ID: "res-1"},
			&Reservation{ID: "res-2"},
		}, nil) // return nil error so it tries again

	store.On("Poll", nodeID, false, mock.Anything).
		Return([]*Reservation{
			&Reservation{ID: "res-3"},
			&Reservation{ID: "res-4"},
		}, ErrPollEOS)

	reservations := []*Reservation{}
	for res := range chn {
		reservations = append(reservations, res)
	}

	require.Len(reservations, 4)
	require.Equal("res-1", reservations[0].ID)
	require.Equal("res-2", reservations[1].ID)
	require.Equal("res-3", reservations[2].ID)
	require.Equal("res-4", reservations[3].ID)
}
