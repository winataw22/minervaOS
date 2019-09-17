package provision

import (
	"context"
	"testing"
	"time"

	"github.com/threefoldtech/zosv2/modules"
)

type testReservationStore struct {
	fReserve func(r Reservation, nodeID modules.Identifier) error
	fPoll    func(nodeID modules.Identifier, all bool, since time.Time) ([]*Reservation, error)
	fGet     func(id string) (Reservation, error)
}

func (s testReservationStore) Reserve(r Reservation, nodeID modules.Identifier) error {
	return s.fReserve(r, nodeID)
}
func (s testReservationStore) Poll(nodeID modules.Identifier, all bool, since time.Time) ([]*Reservation, error) {
	return s.fPoll(nodeID, all, since)
}
func (s testReservationStore) Get(id string) (Reservation, error) {
	return s.fGet(id)
}

func TestHTTPReservationSource(t *testing.T) {
	t.Skip()
	store := testReservationStore{
		fPoll: func(nodeID modules.Identifier, all bool, since time.Time) ([]*Reservation, error) {
			time.Sleep(time.Second * 10)
			return []*Reservation{}, nil
		},
	}

	source := HTTPSource(store, modules.StrIdentifier("nodeID"))
	c := source.Reservations(context.TODO())
	_ = <-c
	// assert.Equal(t, 0, len(result))
}
