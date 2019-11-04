package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/threefoldtech/zos/pkg/provision"
)

type reservation struct {
	Reservation *provision.Reservation `json:"reservation"`
	Result      *provision.Result      `json:"result"`
	Deleted     bool                   `json:"deleted"`
	NodeID      string                 `json:"node_id"`
}

type reservationsStore struct {
	Reservations []*reservation `json:"reservations"`
	m            sync.RWMutex
}

func loadProvisionStore() (*reservationsStore, error) {
	store := &reservationsStore{
		Reservations: []*reservation{},
	}
	f, err := os.OpenFile("reservations.json", os.O_RDONLY, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		return store, err
	}
	return store, nil
}

func (s *reservationsStore) Save() error {
	s.m.RLock()
	defer s.m.RUnlock()

	f, err := os.OpenFile("reservations.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *reservationsStore) List() []*reservation {
	s.m.RLock()
	defer s.m.RUnlock()
	out := make([]*reservation, len(s.Reservations))

	copy(out, s.Reservations)
	return out
}

func (s *reservationsStore) Get(ID string) (*reservation, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	for _, r := range s.Reservations {
		if r.Reservation.ID == ID {
			return r, nil
		}
	}

	return nil, fmt.Errorf("reservation %s not found", ID)
}

func (s *reservationsStore) Add(nodeID string, res *provision.Reservation) error {
	s.m.Lock()
	defer s.m.Unlock()
	res.ID = fmt.Sprintf("%d-1", len(s.Reservations))
	s.Reservations = append(s.Reservations, &reservation{
		NodeID:      nodeID,
		Reservation: res,
	})
	return nil
}

func (s *reservationsStore) GetReservations(nodeID string, from uint64) []*provision.Reservation {
	output := []*provision.Reservation{}

	s.m.RLock()
	defer s.m.RUnlock()

	for _, r := range s.Reservations {
		// skip reservation aimed at another node
		if r.NodeID != nodeID {
			continue
		}

		resID, _, err := r.Reservation.SplitID()
		if err != nil {
			continue
		}

		if from == 0 ||
			(!r.Reservation.Expired() && resID >= from) ||
			(r.Reservation.ToDelete && !r.Deleted) {
			output = append(output, r.Reservation)
		}
	}

	return output
}
