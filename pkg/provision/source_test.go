package provision

import (
	"context"
	"testing"
	"time"

	"github.com/threefoldtech/zos/pkg"
)

type testReservationStore struct {
	fReserve func(r Reservation, nodeID pkg.Identifier) error
	fPoll    func(nodeID pkg.Identifier, all bool, since time.Time) ([]*Reservation, error)
	fGet     func(id string) (Reservation, error)
}

func (s testReservationStore) Reserve(r Reservation, nodeID pkg.Identifier) error {
	return s.fReserve(r, nodeID)
}
func (s testReservationStore) Poll(nodeID pkg.Identifier, all bool, since time.Time) ([]*Reservation, error) {
	return s.fPoll(nodeID, all, since)
}
func (s testReservationStore) Get(id string) (Reservation, error) {
	return s.fGet(id)
}

func TestHTTPReservationSource(t *testing.T) {
	t.Skip()
	store := testReservationStore{
		fPoll: func(nodeID pkg.Identifier, all bool, since time.Time) ([]*Reservation, error) {
			time.Sleep(time.Second * 10)
			return []*Reservation{}, nil
		},
	}

	source := HTTPSource(store, pkg.StrIdentifier("nodeID"))
	c := source.Reservations(context.TODO())
	_ = <-c
	// assert.Equal(t, 0, len(result))
}
