package provision

import (
	"fmt"

	"github.com/pkg/errors"
)

// ReservationGetter define the interface how to get
// a reservation from its ID
type ReservationGetter interface {
	Get(id string) (*Reservation, error)
}

// OwnerCache allows to get the user ID of owner of a reservation
type OwnerCache struct {
	local  ReservationGetter
	remote ReservationGetter
}

// NewCache returns a new initialized reservation cache
func NewCache(local, remote ReservationGetter) *OwnerCache {
	return &OwnerCache{
		local:  local,
		remote: remote,
	}
}

// OwnerOf return the userID of the creator of the reservation
// identified by reservationID
func (c *OwnerCache) OwnerOf(reservationID string) (string, error) {
	var (
		r   *Reservation
		err error
	)

	for _, source := range []ReservationGetter{c.local, c.remote} {
		if source == nil {
			continue
		}
		r, err = c.local.Get(reservationID)
		if err == nil {
			break
		}
	}

	if r == nil {
		return "", fmt.Errorf("failed to get owner of reservation %s", reservationID)
	}

	if err := Verify(r); err != nil {
		return "", errors.Wrapf(err, "failed to get owner of reservation %s", reservationID)
	}

	return r.User, nil
}
