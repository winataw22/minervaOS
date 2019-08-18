package provision

import (
	"github.com/golang/groupcache/lru"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Cache is used to keep a mapping
// between a reservation ID and its owner
type Cache struct {
	cache *lru.Cache
	store ReservationStore
}

// NewCache returns a new initialized reservation cache
func NewCache(cacheSize int, provisionURL string) *Cache {
	cache := lru.New(cacheSize)
	cache.OnEvicted = func(key lru.Key, _ interface{}) {
		log.Info().Msgf("key %s removed from cache", key)
	}
	return &Cache{
		cache: cache,
		store: NewhHTTPStore(provisionURL),
	}
}

// OwnerOf return the userID of the creator of the reservation
// identified by reservationID
func (c *Cache) OwnerOf(reservationID string) (string, error) {
	var owner string

	result, exist := c.cache.Get(reservationID)
	if exist {
		owner = result.(string)
	} else {
		r, err := c.store.Get(reservationID)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get owner of reservation %s", reservationID)
		}

		if err := Verify(r); err != nil {
			return "", errors.Wrapf(err, "failed to get owner of reservation %s", reservationID)
		}
		owner = r.User
		c.cache.Add(reservationID, owner)
	}

	return owner, nil
}
