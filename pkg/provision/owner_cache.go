package provision

import (
	"fmt"
)

// ReservationGetter define the interface how to get
// a reservation from its ID
type ReservationGetter interface {
	Get(id string) (*Reservation, error)
}

// ownerCache allows to get the user ID of owner of a reservation
type ownerCache struct {
	local  ReservationGetter
	remote ReservationGetter
}

// NewCache returns a new initialized reservation cache
func NewCache(local, remote ReservationGetter) OwnerCache {
	return &ownerCache{
		local:  local,
		remote: remote,
	}
}

// OwnerOf return the userID of the creator of the reservation
// identified by reservationID
func (c *ownerCache) OwnerOf(reservationID string) (string, error) {
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

	return r.User, nil
}
