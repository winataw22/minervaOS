package provision

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/versioned"
)

// Counter interface
type Counter interface {
	// Increment counter atomically by one
	Increment() int64
	// Decrement counter atomically by one
	Decrement() int64
	// Current returns the current value
	Current() int64
}

type counterNop struct{}

func (c *counterNop) Increment() int64 {
	return 0
}

func (c *counterNop) Decrement() int64 {
	return 0
}

func (c *counterNop) Current() int64 {
	return 0
}

var (
	nop counterNop
)

// counterImpl value for safe increment/decrement
type counterImpl int64

// Increment counter atomically by one
func (c *counterImpl) Increment() int64 {
	return atomic.AddInt64((*int64)(c), 1)
}

// Decrement counter atomically by one
func (c *counterImpl) Decrement() int64 {
	return atomic.AddInt64((*int64)(c), -1)
}

// Current returns the current value
func (c *counterImpl) Current() int64 {
	return atomic.LoadInt64((*int64)(c))
}

// FSStore is a in reservation store
// using the filesystem as backend
type FSStore struct {
	sync.RWMutex
	root string

	counters struct {
		containers counterImpl
		volumes    counterImpl
		networks   counterImpl
		zdb        counterImpl
		vm         counterImpl
		debug      counterImpl
	}
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*FSStore, error) {
	if err := os.RemoveAll(root); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}
	return &FSStore{
		root: root,
	}, nil
}

// Counters returns stats about the cashed reservations
func (s *FSStore) Counters() pkg.ProvisionCounters {
	return pkg.ProvisionCounters{
		Container: s.counters.containers.Current(),
		Volume:    s.counters.volumes.Current(),
		Network:   s.counters.networks.Current(),
		ZDB:       s.counters.zdb.Current(),
		VM:        s.counters.vm.Current(),
		Debug:     s.counters.debug.Current(),
	}
}

func (s *FSStore) counterFor(typ ReservationType) Counter {
	switch typ {
	case ContainerReservation:
		return &s.counters.containers
	case VolumeReservation:
		return &s.counters.volumes
	case NetworkReservation:
		return &s.counters.networks
	case ZDBReservation:
		return &s.counters.zdb
	case DebugReservation:
		return &s.counters.debug
	case KubernetesReservation:
		return &s.counters.vm
	default:
		// this will avoid nil pointer
		return &nop
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
	writer, err := versioned.NewWriter(f, reservationSchemaLastVersion)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(writer).Encode(r); err != nil {
		return err
	}

	s.counterFor(r.Type).Increment()

	return nil
}

// Remove a reservation from the store
func (s *FSStore) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	r, err := s.get(id)
	if os.IsNotExist(errors.Cause(err)) {
		return nil
	}

	path := filepath.Join(s.root, id)
	err = os.Remove(path)
	if os.IsNotExist(err) {
		// shouldn't happen because we just did a get
		return nil
	} else if err != nil {
		return err
	}

	s.counterFor(r.Type).Decrement()

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
		if r.Expired() {
			r.Tag = Tag{"source": "FSStore"}
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

	return s.get(id)
}

// Exists checks if the reservation ID is in the store
func (s *FSStore) Exists(id string) (bool, error) {
	s.RLock()
	defer s.RUnlock()

	path := filepath.Join(s.root, id)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *FSStore) get(id string) (*Reservation, error) {
	path := filepath.Join(s.root, id)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "reservation %s not found", id)
	} else if err != nil {
		return nil, err
	}

	defer f.Close()
	reader, err := versioned.NewReader(f)
	if versioned.IsNotVersioned(err) {
		if _, err := f.Seek(0, 0); err != nil { // make sure to read from start
			return nil, err
		}
		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), f)
	}

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", reservationSchemaV1))
	var reservation Reservation

	if validV1(reader.Version()) {
		if err := json.NewDecoder(reader).Decode(&reservation); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown reservation object version (%s)", reader.Version())
	}
	reservation.Tag = Tag{"source": "FSStore"}
	return &reservation, nil
}

// Close makes sure the backend of the store is closed properly
func (s *FSStore) Close() error {
	return nil
}
