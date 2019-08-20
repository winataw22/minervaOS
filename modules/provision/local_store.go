package provision

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// FSStore is a in reservation store
// using the filesystem as backend
type FSStore struct {
	sync.RWMutex
	root string
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) *FSStore {
	return &FSStore{
		root: root,
	}
}

// Add a reservation to the store
func (s *FSStore) Add(r *Reservation) error {
	s.Lock()
	defer s.Unlock()

	// ensure direcory exists
	if err := os.MkdirAll(s.root, 0770); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.root, r.ID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("reservation %s already in the store", r.ID)
		}
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(r); err != nil {
		return err
	}

	return nil
}

// Remove a reservation from the store
func (s *FSStore) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	path := filepath.Join(s.root, id)
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *FSStore) GetExpired() ([]*Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	rs := make([]*Reservation, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		r, err := s.Get(info.Name())
		if err != nil {
			return nil, err
		}
		if r.expired() {
			rs = append(rs, r)
		}

	}

	return rs, nil
}

// Get retrieves a specific reservation using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *FSStore) Get(id string) (*Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	path := filepath.Join(s.root, id)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("reservation %s not found", id)
		}
		return nil, err
	}
	defer f.Close()

	r := &Reservation{}
	if err := json.NewDecoder(f).Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

// Close makes sure the backend of the store is closed properly
func (s *FSStore) Close() error {
	return nil
}
