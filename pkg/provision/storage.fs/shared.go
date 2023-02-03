package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Shared, returns a workload id that reference the workload with that
// name. The workload type must be of a `Shared` type.
// A Shared workload type means that the workload (of that type) can be
// accessed by other deployments for the same twin. A shared workload
// should be only updatable only via the deployment that creates it
func (s *Fs) GetShared(twinID uint32, name gridtypes.Name) (gridtypes.WorkloadID, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.shared(twinID, name)
}

func (s *Fs) SharedByTwin(twinID uint32) ([]gridtypes.WorkloadID, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.sharedByTwin(twinID)
}

func (s *Fs) sharedByTwin(twinID uint32) ([]gridtypes.WorkloadID, error) {
	root := filepath.Join(s.root, sharedSubDir, fmt.Sprint(twinID))
	infos, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to list shared user workloads")
	}
	var ids []gridtypes.WorkloadID
	for _, entry := range infos {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if info.Mode().Type() != fs.ModeSymlink {
			log.Warn().
				Uint32("twin", twinID).
				Str("name", info.Name()).
				Msg("found non symlink file in twin shared workloads")
			continue
		}

		id, err := s.shared(twinID, gridtypes.Name(info.Name()))
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}
func (s *Fs) shared(twinID uint32, name gridtypes.Name) (id gridtypes.WorkloadID, err error) {
	link := s.rooted(s.sharedLinkPath(twinID, name))
	target, err := os.Readlink(link)
	if err != nil {
		return id, errors.Wrapf(err, "failed to read link to deployment from '%s'", link)
	}
	// target has base name as the 'contract id'
	dl, err := strconv.ParseUint(filepath.Base(target), 10, 32)
	if err != nil {
		return id, errors.Wrapf(err, "invalid link '%s' to target '%s'", link, target)
	}

	return gridtypes.NewUncheckedWorkloadID(twinID, dl, name), nil
}

func (s *Fs) sharedCreate(d *gridtypes.Deployment, name gridtypes.Name) error {
	target := s.rooted(s.deploymentPath(d))
	src := s.rooted(s.sharedLinkPath(d.TwinID, name))

	dir := filepath.Dir(src)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "failed to create shared twin directory")
	}

	target, err := filepath.Rel(dir, target)
	if err != nil {
		return err
	}

	return os.Symlink(target, src)
}

func (s *Fs) sharedDelete(d *gridtypes.Deployment, name gridtypes.Name) error {
	src := s.rooted(s.sharedLinkPath(d.TwinID, name))
	return os.Remove(src)
}
