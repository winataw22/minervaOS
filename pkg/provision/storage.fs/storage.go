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

	sharedSubDir = "shared"
)

// Fs is a in reservation cache using the filesystem as backend
type Fs struct {
	m    sync.RWMutex
	root string
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*Fs, error) {
	store := &Fs{
		root: root,
	}

	for _, dir := range []string{sharedSubDir} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0770); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *Fs) sharedLinkPath(twinID uint32, name gridtypes.Name) string {
	return filepath.Join(sharedSubDir, fmt.Sprint(twinID), string(name))
}

func (s *Fs) deploymentPath(d *gridtypes.Deployment) string {
	return filepath.Join(fmt.Sprint(d.TwinID), fmt.Sprint(d.ContractID))
}

func (s *Fs) rooted(p ...string) string {
	return filepath.Join(s.root, filepath.Join(p...))
}

// Delete is only used for the migration
func (s *Fs) Delete(d gridtypes.Deployment) error {
	s.m.Lock()
	defer s.m.Unlock()

	path := s.rooted(s.deploymentPath(&d))
	return os.RemoveAll(path)
}

// Add workload to database
func (s *Fs) Add(d gridtypes.Deployment) error {
	s.m.Lock()
	defer s.m.Unlock()

	path := s.rooted(s.deploymentPath(&d))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "failed to crate directory")
	}

	// make sure that this deployment does not actually
	// redefine a "sharable" workload.
	for _, wl := range d.GetShareables() {
		conflict, err := s.shared(d.TwinID, wl.Name)
		if err == nil {
			return errors.Wrapf(
				provision.ErrDeploymentConflict,
				"sharable workload '%s' is conflicting with another workload '%s'",
				string(wl.Name), conflict)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.Wrap(err, "failed to check conflicts")
		}
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

	// make sure that this deployment does not actually
	// redefine a "sharable" workload.
	for _, wl := range d.GetShareables() {
		if err := s.sharedCreate(&d, wl.Name); err != nil {
			return errors.Wrap(err, "failed to store sharable workloads")
		}
	}

	return nil
}

// Set updates value of a workload, the reservation must exists
// otherwise an error is returned
func (s *Fs) Set(dl gridtypes.Deployment) error {
	s.m.Lock()
	defer s.m.Unlock()

	sharedIDs, err := s.sharedByTwin(dl.TwinID)
	if err != nil {
		return errors.Wrap(err, "failed to get all sharable user workloads")
	}

	this := map[gridtypes.Name]gridtypes.WorkloadID{}
	taken := map[gridtypes.Name]gridtypes.WorkloadID{}
	for _, shared := range sharedIDs {
		_, contract, name, _ := shared.Parts()
		if contract == dl.ContractID {
			this[name] = shared
		} else {
			taken[name] = shared
		}
	}

	// does this workload defines a new sharable workload. In that case
	// we need to make sure that this does not conflict with the current
	// set of twin sharable workloads. but should not conflict with itself
	for _, wl := range dl.GetShareables() {
		if conflict, ok := taken[wl.Name]; ok {
			return errors.Wrapf(
				provision.ErrDeploymentConflict,
				"sharable workload '%s' is conflicting with another workload '%s'",
				string(wl.Name), conflict)
		}
	}

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

	// now we make sure that all sharable (and active) workloads
	// on this deployment is referenced correctly
	var tolink []gridtypes.Name
	for _, wl := range dl.GetShareables() {
		// if workload result is not set yet. or if the state is OK
		// it means the workload still need to be treated as shared object
		if wl.Result.IsNil() || wl.Result.State == gridtypes.StateOk {
			// workload with no results, so we should keep the link
			if _, ok := this[wl.Name]; ok {
				// avoid unlinking
				delete(this, wl.Name)
			} else {
				// or if new, add to tolink
				tolink = append(tolink, wl.Name)
			}
		}
		// if result state is set to anything else (deleted, or error)
		// we then leave it in the `this` map which means they
		// get to be unlinked. so the name can be used again by the same twin
	}
	// we either removed the names that should be kept (state = ok or no result yet)
	// and new ones have been added to tolink. so it's safe to clean up all links
	// in this.
	for name := range this {
		if err := s.sharedDelete(&dl, name); err != nil {
			log.Error().Err(err).
				Uint64("contract", dl.ContractID).
				Stringer("name", name).
				Msg("failed to clean up shared workload '%d.%s'")
		}
	}

	for _, new := range tolink {
		if err := s.sharedCreate(&dl, new); err != nil {
			return err
		}
	}

	return nil
}

// Get gets a workload by id
func (s *Fs) get(path string) (gridtypes.Deployment, error) {
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
	s.m.RLock()
	defer s.m.RUnlock()

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
		if !entry.IsDir() || entry.Name() == sharedSubDir {
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
