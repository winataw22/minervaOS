package provision

import (
	"fmt"
	"sync"
)

// MemStore is a in memory reservation store
type MemStore struct {
	sync.RWMutex
	m map[string]*Reservation
}

// NewMemStore creates a in memory reservation store
func NewMemStore() *MemStore {
	return &MemStore{
		m: make(map[string]*Reservation),
	}
}

// Add a reservation ID to the store
func (s *MemStore) Add(r *Reservation) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[r.ID]; ok {
		return fmt.Errorf("reservation %s already in the store", r.ID)
	}

	s.m[r.ID] = r
	return nil
}

// Remove a reservation ID from the store
func (s *MemStore) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[id]; ok {
		delete(s.m, id)
	}

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *MemStore) GetExpired() ([]*Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	output := make([]*Reservation, 0, len(s.m)/2)
	for _, r := range s.m {
		if !r.expired() {
			continue
		}
		output = append(output, r)
	}
	return output, nil
}

// Get retrieves a specific reservation using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *MemStore) Get(id string) (*Reservation, error) {
	s.Lock()
	defer s.Unlock()

	r, ok := s.m[id]
	if !ok {
		return nil, fmt.Errorf("reservation not found")
	}
	return r, nil
}

// Close makes sure the backend of the store is closed properly
func (s *MemStore) Close() error {
	return nil
}
