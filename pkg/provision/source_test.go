package provision

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

type TestPollSource struct {
	mock.Mock
}

func (s *TestPollSource) Poll(nodeID pkg.Identifier, from uint64) ([]*Reservation, error) {
	returns := s.Called(nodeID, from)
	return returns.Get(0).([]*Reservation), returns.Error(1)
}

func TestHTTPReservationSource(t *testing.T) {
	require := require.New(t)
	var store TestPollSource

	nodeID := pkg.StrIdentifier("node-id")
	source := PollSource(&store, nodeID)
	chn := source.Reservations(context.Background())

	store.On("Poll", nodeID, uint64(0)).
		Return([]*Reservation{
			&Reservation{ID: "1-1"},
			&Reservation{ID: "1-2"},
		}, ErrPollEOS)

	reservations := []*Reservation{}
	for res := range chn {
		reservations = append(reservations, res)
	}

	require.Len(reservations, 2)
	require.Equal("1-1", reservations[0].ID)
	require.Equal("1-2", reservations[1].ID)
}

func TestHTTPReservationSourceMultiple(t *testing.T) {
	require := require.New(t)
	var store TestPollSource

	nodeID := pkg.StrIdentifier("node-id")
	source := PollSource(&store, nodeID)

	// don't delay the test for long
	source.(*pollSource).maxSleep = 500 * time.Microsecond

	chn := source.Reservations(context.Background())

	store.On("Poll", nodeID, uint64(0)).
		Return([]*Reservation{
			&Reservation{ID: "1-1"},
			&Reservation{ID: "2-1"},
		}, nil) // return nil error so it tries again

	store.On("Poll", nodeID, uint64(3)).
		Return([]*Reservation{
			&Reservation{ID: "3-1"},
			&Reservation{ID: "4-1"},
		}, ErrPollEOS)

	reservations := []*Reservation{}
	for res := range chn {
		reservations = append(reservations, res)
	}

	require.Len(reservations, 4)
	require.Equal("1-1", reservations[0].ID)
	require.Equal("2-1", reservations[1].ID)
	require.Equal("3-1", reservations[2].ID)
	require.Equal("4-1", reservations[3].ID)
}

type TestTrackSource struct {
	Max   uint64
	ID    uint64
	Calls []int64
}

func (s *TestTrackSource) Poll(nodeID pkg.Identifier, from uint64) ([]*Reservation, error) {
	if s.ID == s.Max {
		return nil, ErrPollEOS
	}

	s.Calls = append(s.Calls, time.Now().Unix())

	defer func() {
		s.ID++
	}()

	return []*Reservation{
		&Reservation{
			ID: fmt.Sprint(s.ID, "-", "0"),
		},
	}, nil
}

func TestHTTPReservationSourceSleep(t *testing.T) {
	require := require.New(t)
	store := TestTrackSource{
		Max: 4,
	}

	nodeID := pkg.StrIdentifier("node-id")
	source := PollSource(&store, nodeID)

	// don't delay the test for long
	source.(*pollSource).maxSleep = 3 * time.Second

	chn := source.Reservations(context.Background())

	reservations := []*Reservation{}
	for res := range chn {
		reservations = append(reservations, res)
	}

	require.Len(reservations, 4)
	require.Equal("0-0", reservations[0].ID)
	require.Equal("1-0", reservations[1].ID)
	require.Equal("2-0", reservations[2].ID)
	require.Equal("3-0", reservations[3].ID)

	require.Len(store.Calls, 4)
	for i := 1; i < len(store.Calls); i++ {
		require.Equal(int64(3), store.Calls[i]-store.Calls[i-1])
	}
}
