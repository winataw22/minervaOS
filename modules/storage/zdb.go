package storage

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage/filesystem"
)

// Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
// of specified size, type and mode
// it returns the volume ID and its path or an error if it couldn't allocate enough storage
func (s *storageModule) Allocate(diskType modules.DeviceType, size uint64, mode modules.ZDBMode) (string, string, error) {
	// try to find an existing zdb volume that has still enough storage available
	// if we find it, grow the quota by the requested size
	// if we don't, pick a new pool and create a zdb volume on it with the requested size
	slog := log.With().
		Str("type", string(diskType)).
		Uint64("size", size).
		// Str("mode", string(mode)). TODO: currently the mode is not used
		Logger()

	slog.Info().Msg("try to allocation space for 0-DB")

	for _, pool := range s.volumes {

		// skip pool with wrong disk type
		if pool.Type() != diskType {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {

			// skip all non-zdb volume
			if !strings.HasPrefix(volume.Name(), zdbPoolPrefix) {
				continue
			}

			usage, err := pool.Usage()
			if err != nil {
				return "", "", errors.Wrapf(err, "failed to read usage of volume %s", volume.Name())
			}

			// skip pool with not enough space
			reserved, err := pool.Reserved()
			if err != nil {
				return "", "", errors.Wrapf(err, "failed to read reserved size of pool %s", pool.Name())
			}

			// Make sure adding this filesystem would not bring us over the disk limit
			if reserved+size > usage.Size {
				slog.Info().Msgf("Disk does not have enough space left to hold filesystem")
				continue
			}

			// existing volume with enough grow its limit
			if err := volume.Limit(usage.Size + size); err != nil {
				return "", "", errors.Wrapf(err, "failed to grow limit of volume %s", volume.Name())
			}
			slog.Info().
				Str("volume", volume.Name()).
				Str("path", volume.Path()).
				Msg("space allocated")
			return volume.Name(), volume.Path(), nil
		}
	}

	name, err := genZDBPoolName()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate new sub-volume name")
	}

	volume, err := s.createSubvol(size, name, diskType)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create sub-volume")
	}

	slog.Info().
		Str("volume", volume.Name()).
		Str("path", volume.Path()).
		Msg("space allocated")
	return volume.Name(), volume.Path(), nil
}

// Claim let the system claim the allocated storage used by a 0-db namespace
func (s *storageModule) Claim(name string, size uint64) error {
	var (
		v   filesystem.Volume
		err error
	)

	for _, pool := range s.volumes {

		volumes, err := pool.Volumes()
		if err != nil {
			return errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {
			if volume.Name() == name {
				v = volume
				break
			}
		}

		if v != nil {
			break
		}
	}

	if v == nil {
		return fmt.Errorf("volume named %s not found", name)
	}

	usage, err := v.Usage()
	if err != nil {
		return errors.Wrapf(err, "failed to read usage of volume %s", v.Name())
	}

	// shrink the limit
	if err := v.Limit(usage.Size - size); err != nil {
		return errors.Wrapf(err, "failed to grow limit of volume %s", v.Name())
	}

	return nil
}

const zdbPoolPrefix = "zdb"

func genZDBPoolName() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	name := zdbPoolPrefix + id.String()
	return name, nil
}
