package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/versioned"
)

var (
	// deploymentSchemaV1 reservation schema version 1
	deploymentSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	deploymentSchemaLastVersion = deploymentSchemaV1
)

// Fs is a in reservation cache using the filesystem as backend
type Fs struct {
	m    sync.RWMutex
	root string
}

var _ (provision.Storage) = (*Fs)(nil)

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*Fs, error) {
	store := &Fs{
		root: root,
	}

	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Fs) deploymentPath(d *gridtypes.Deployment) string {
	return filepath.Join(fmt.Sprint(d.TwinID), fmt.Sprint(d.ContractID))
}

func (s *Fs) rooted(p ...string) string {
	return filepath.Join(s.root, filepath.Join(p...))
}

// Add workload to database
func (s *Fs) Add(d gridtypes.Deployment) error {
	s.m.Lock()
	defer s.m.Unlock()

	path := s.rooted(s.deploymentPath(&d))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "failed to crate directory")
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_EXCL,
		0644,
	)

	if os.IsExist(err) {
		return errors.Wrapf(provision.ErrDeploymentExists, "object '%d' exist", d.ContractID)
	} else if err != nil {
		return errors.Wrap(err, "failed to open workload file")
	}

	defer file.Close()
	writer, err := versioned.NewWriter(file, deploymentSchemaLastVersion)
	if err != nil {
		return errors.Wrap(err, "failed to create versioned writer")
	}

	if err := json.NewEncoder(writer).Encode(d); err != nil {
		return errors.Wrap(err, "failed to write workload data")
	}

	return nil
}

// Set updates value of a workload, the reservation must exists
// otherwise an error is returned
func (s *Fs) Set(dl gridtypes.Deployment) error {
	s.m.Lock()
	defer s.m.Unlock()

	path := s.rooted(s.deploymentPath(&dl))
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_TRUNC,
		0644,
	)
	if os.IsNotExist(err) {
		return errors.Wrapf(provision.ErrDeploymentNotExists, "deployment '%d:%d' does not exist", dl.TwinID, dl.ContractID)
	} else if err != nil {
		return errors.Wrap(err, "failed to open workload file")
	}
	defer file.Close()
	writer, err := versioned.NewWriter(file, deploymentSchemaLastVersion)
	if err != nil {
		return errors.Wrap(err, "failed to create versioned writer")
	}

	if err := json.NewEncoder(writer).Encode(dl); err != nil {
		return errors.Wrap(err, "failed to write workload data")
	}

	return nil
}

// Get gets a workload by id
func (s *Fs) get(path string) (gridtypes.Deployment, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	var wl gridtypes.Deployment
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return wl, errors.Wrapf(provision.ErrDeploymentNotExists, "deployment '%s' does not exist", path)
	} else if err != nil {
		return wl, errors.Wrap(err, "failed to open workload file")
	}
	defer file.Close()
	reader, err := versioned.NewReader(file)
	if err != nil {
		return wl, errors.Wrap(err, "failed to load workload")
	}
	version := reader.Version()
	if !version.EQ(deploymentSchemaV1) {
		return wl, fmt.Errorf("invalid workload version")
	}

	if err := json.NewDecoder(reader).Decode(&wl); err != nil {
		return wl, errors.Wrap(err, "failed to read workload data")
	}

	return wl, nil
}

// Get gets a workload by id
func (s *Fs) Get(twin uint32, deployment uint64) (gridtypes.Deployment, error) {
	path := s.rooted(filepath.Join(fmt.Sprint(twin), fmt.Sprint(deployment)))

	return s.get(path)
}

// ByTwin return list of deployment for given twin id
func (s *Fs) ByTwin(twin uint32) ([]uint64, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.byTwin(twin)
}

func (s *Fs) byTwin(twin uint32) ([]uint64, error) {
	base := filepath.Join(s.root, fmt.Sprint(twin))

	entities, err := ioutil.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to list twin directory")
	}
	ids := make([]uint64, 0, len(entities))
	for _, entry := range entities {
		if entry.IsDir() {
			continue
		}

		id, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			log.Error().Str("name", entry.Name()).Err(err).Msg("invalid deployment id file")
			continue
		}

		ids = append(ids, uint64(id))
	}

	return ids, nil
}

// Twins lists available users
func (s *Fs) Twins() ([]uint32, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.twins()
}

func (s *Fs) twins() ([]uint32, error) {
	entities, err := ioutil.ReadDir(s.root)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to list twins directory")
	}
	ids := make([]uint32, 0, len(entities))
	for _, entry := range entities {
		if !entry.IsDir() {
			continue
		}

		id, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			log.Error().Str("name", entry.Name()).Err(err).Msg("invalid twin id directory, removing")
			os.RemoveAll(filepath.Join(s.root, entry.Name()))
			continue
		}

		ids = append(ids, uint32(id))
	}

	return ids, nil
}

// Capacity returns the total capacity of all deployments
// that are in OK state.
func (s *Fs) Capacity() (cap gridtypes.Capacity, err error) {
	s.m.RLock()
	defer s.m.RUnlock()

	twins, err := s.twins()
	if err != nil {
		return cap, err
	}

	for _, twin := range twins {
		ids, err := s.byTwin(twin)
		if err != nil {
			return cap, err
		}

		for _, id := range ids {
			p := s.rooted(fmt.Sprint(twin), fmt.Sprint(id))
			deployment, err := s.get(p)
			if err != nil {
				return cap, err
			}

			for _, wl := range deployment.Workloads {
				if wl.Result.State != gridtypes.StateOk {
					continue
				}

				c, err := wl.Capacity()
				if err != nil {
					return cap, err
				}

				cap.Add(&c)
			}
		}
	}

	return
}

// Close makes sure the backend of the store is closed properly
func (s *Fs) Close() error {
	return nil
}
