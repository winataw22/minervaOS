package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
	"github.com/threefoldtech/zos/pkg/versioned"
)

var (
	// reservationSchemaV1 reservation schema version 1
	reservationSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	reservationSchemaLastVersion = reservationSchemaV1
)

// Fs is a in reservation cache using the filesystem as backend
type Fs struct {
	sync.RWMutex
	root string
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*Fs, error) {
	store := &Fs{
		root: root,
	}
	if app.IsFirstBoot("provisiond") {
		log.Info().Msg("first boot, empty reservation cache")
		if err := store.removeAllButPersistent(root); err != nil {
			return nil, err
		}

		if err := app.MarkBooted("provisiond"); err != nil {
			return nil, errors.Wrap(err, "fail to mark provisiond as booted")
		}
	}

	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}

	log.Info().Msg("restart detected, keep reservation cache intact")

	return store, nil
}

//TODO: i think both sync and removeAllButPersistent can be merged into
// one method because now it scans the same directory twice.
func (s *Fs) removeAllButPersistent(rootPath string) error {
	// if rootPath is not present on the filesystem, return
	_, err := os.Stat(rootPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, r error) error {
		if r != nil {
			return r
		}
		// if a file with size 0 is present we can assume its empty and remove it
		if info.Size() == 0 {
			log.Warn().Str("filename", info.Name()).Msg("cached reservation %d found, but file is empty, removing.")
			return os.Remove(path)
		}

		if info.IsDir() {
			return nil
		}
		reservationType, err := s.getType(filepath.Base(path))
		if err != nil {
			return err
		}
		if reservationType != primitives.VolumeReservation {
			log.Info().Msgf("Removing %s from cache", path)
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Error().Msgf("error walking the path %q: %v\n", rootPath, err)
		return err
	}
	return nil
}

// Sync update the statser with all the reservation present in the cache
func (s *Fs) Sync(statser provision.Statser) error {
	//this should probably be reversed and moved to the Statser object instead

	s.RLock()
	defer s.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		r, err := s.get(info.Name())
		if err != nil {
			return err
		}

		if !r.Expired() {
			if err := statser.Increment(r); err != nil {
				return fmt.Errorf("fail to update stats:%w", err)
			}
		}
	}

	return nil
}

// Add a reservation to the store
func (s *Fs) Add(r *provision.Reservation) error {
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

	return nil
}

// Remove a reservation from the store
func (s *Fs) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	path := filepath.Join(s.root, id)
	err := os.Remove(path)
	if os.IsNotExist(errors.Cause(err)) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *Fs) GetExpired() ([]*provision.Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	rs := make([]*provision.Reservation, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		// if the file is empty, remove it and return.
		if info.Size() == 0 {
			if info.Size() == 0 {
				log.Warn().Str("filename", info.Name()).Msg("cached reservation %d found, but file is empty, removing.")
				return nil, os.Remove(path.Join(s.root, info.Name()))
			}
		}

		r, err := s.get(info.Name())
		if err != nil {
			return nil, err
		}
		if r.Expired() {
			// r.Tag = Tag{"source": "FSStore"}
			rs = append(rs, r)
		}

	}

	return rs, nil
}

// Get retrieves a specific reservation using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *Fs) Get(id string) (*provision.Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	return s.get(id)
}

// getType retrieves a specific reservation's type using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *Fs) getType(id string) (provision.ReservationType, error) {
	r, err := s.get(id)
	if err != nil {
		return provision.ReservationType(0), err
	}
	return r.Type, nil
}

// Exists checks if the reservation ID is in the store
func (s *Fs) Exists(id string) (bool, error) {
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

func (s *Fs) get(id string) (*provision.Reservation, error) {
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
	var reservation provision.Reservation

	if validV1(reader.Version()) {
		if err := json.NewDecoder(reader).Decode(&reservation); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown reservation object version (%s)", reader.Version())
	}
	// reservation.Tag = Tag{"source": "FSStore"}
	return &reservation, nil
}

// Close makes sure the backend of the store is closed properly
func (s *Fs) Close() error {
	return nil
}
