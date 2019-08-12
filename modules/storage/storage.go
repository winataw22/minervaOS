package storage

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage/filesystem"
)

const (
	cacheTarget = "/var/cache"
	cacheLabel  = "zos-cache"
	cacheSize   = 20 * 1024 * 1024 * 1024 // 20GB
)

var (
	diskBase = map[modules.RaidProfile]int{
		modules.Single: 1,
		modules.Raid1:  2,
		modules.Raid10: 4,
	}
)

type storageModule struct {
	volumes []filesystem.Pool
	devices filesystem.DeviceManager
}

// New create a new storage module service
func New() (modules.StorageModule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	m, err := filesystem.DefaultDeviceManager(ctx)
	if err != nil {
		return nil, err
	}

	s := &storageModule{
		volumes: []filesystem.Pool{},
		devices: m,
	}

	// go for a simple linear setup right now
	err = s.initialize(modules.StoragePolicy{
		Raid:     modules.Single,
		Disks:    1,
		MaxPools: 0,
	})
	return s, err
}

/**
initialize, must be called at least onetime each boot.
What Initialize will do is the following:
 - Try to mount prepared pools (if they are not mounted already)
 - Scan free devices, apply the policy.
 - If new pools were created, the pool is going to be mounted automatically
**/
func (s *storageModule) initialize(policy modules.StoragePolicy) error {
	log.Info().Msgf("Initializing storage module")
	defer log.Info().Msgf("Finished initializing storage module")

	// Make sure we finish in 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	fs := filesystem.NewBtrfs(s.devices)

	// remount all existing pools
	log.Info().Msgf("Remounting existing volumes")
	log.Debug().Msgf("Searching for existing volumes")
	existingPools, err := fs.List(ctx, filesystem.All)
	if err != nil {
		return err
	}

	for _, volume := range existingPools {
		if _, mounted := volume.Mounted(); mounted {
			log.Debug().Msgf("Volume %s already mounted", volume.Name())
			// volume is already mounted, skip mounting it again
			continue
		}
		_, err = volume.Mount()
		if err != nil {
			return err
		}
		log.Debug().Msgf("Mounted volume %s", volume.Name())
	}
	s.volumes = append(s.volumes, existingPools...)

	// list disks
	log.Info().Msgf("Finding free disks")
	disks, err := s.devices.Devices(ctx)
	if err != nil {
		return err
	}

	// collect free disks, sort by read time so faster disks are first
	sort.Sort(filesystem.ByReadTime(disks))
	freeDisks := filesystem.DeviceCache{}
	for idx := range disks {
		if !disks[idx].Used() {
			log.Debug().Msgf("Found free device %s", disks[idx].Path)
			freeDisks = append(freeDisks, disks[idx])
		}
	}

	log.Info().Msgf("Creating new volumes using policy %s", policy.Raid)

	// sanity check for disk amount
	diskBase, exists := diskBase[policy.Raid]
	if !exists {
		return fmt.Errorf("unrecognized storage policy %s", policy.Raid)
	}
	if int(policy.Disks)%diskBase != 0 {
		return fmt.Errorf("invalid amount of disks (%d) for volume for configuration %v", policy.Disks, policy.Raid)
	}

	// create new pools if applicable
	// for now create as much pools as we can, need to think more about this
	newPools := []filesystem.Pool{}

	// also make sure pools are homogenous, only 1 type of device per pool
	ssds := filesystem.DeviceCache{}
	hdds := filesystem.DeviceCache{}

	for idx := range freeDisks {
		if freeDisks[idx].DiskType == modules.SSDDevice {
			ssds = append(ssds, freeDisks[idx])
		} else {
			hdds = append(hdds, freeDisks[idx])
		}
	}

	createdPools := 0
	fdisks := []filesystem.DeviceCache{ssds, hdds}
	for idx := range fdisks {
		possiblePools := len(fdisks[idx]) / int(policy.Disks)
		// only create up to the specified amount of pools
		if policy.MaxPools != 0 && int(policy.MaxPools) < possiblePools-createdPools {
			possiblePools = int(policy.MaxPools)
		}
		log.Debug().Msgf("Creating %d new volumes", possiblePools)

		for i := 0; i < possiblePools; i++ {
			log.Debug().Msgf("Creating new volume %d", i)
			poolDevices := filesystem.DeviceCache{}

			for j := 0; j < int(policy.Disks); j++ {
				log.Debug().Msgf("Grabbing device %d: %s for new volume", i*int(policy.Disks)+j, fdisks[idx][i*int(policy.Disks)+j].Path)
				poolDevices = append(poolDevices, fdisks[idx][i*int(policy.Disks)+j])
			}

			pool, err := fs.Create(ctx, uuid.New().String(), poolDevices, policy.Raid)
			if err != nil {
				return err
			}

			newPools = append(newPools, pool)
			createdPools++
		}
	}

	// make sure new pools are mounted
	log.Info().Msgf("Making sure new volumes are mounted")
	for idx := range newPools {
		if _, mounted := newPools[idx].Mounted(); !mounted {
			log.Debug().Msgf("Mounting volume %s", newPools[idx].Name())
			if _, err = newPools[idx].Mount(); err != nil {
				return err
			}
		}
	}

	s.volumes = append(s.volumes, newPools...)

	return s.ensureCache()
}

// CreateFilesystem with the given size in a storage pool.
func (s *storageModule) CreateFilesystem(name string, size uint64, poolType modules.DeviceType) (string, error) {
	log.Info().Msgf("Creating new volume with size %d", size)

	fs, err := s.createSubvol(size, name, poolType)
	if err != nil {
		return "", err
	}
	return fs.Path(), nil
}

// ReleaseFilesystem with the given name, this will unmount and then delete
// the filesystem. After this call, the caller must not perform any more actions
// on this filesystem
func (s *storageModule) ReleaseFilesystem(name string) error {
	log.Info().Msgf("Deleting volume %v", name)

	for idx := range s.volumes {
		filesystems, err := s.volumes[idx].Volumes()
		if err != nil {
			return err
		}
		for jdx := range filesystems {
			if filesystems[jdx].Name() == name {
				log.Debug().Msgf("Removing filesystem %v in volume %v", filesystems[jdx].Name(), s.volumes[idx].Name())
				return s.volumes[idx].RemoveVolume(filesystems[jdx].Name())
			}
		}
	}

	log.Warn().Msgf("Could not find filesystem %v", name)
	return nil
}

// Path return the path of the mountpoint of the named filesystem
// if no volume with name exists, an empty path and an error is returned
func (s *storageModule) Path(name string) (string, error) {
	for idx := range s.volumes {
		filesystems, err := s.volumes[idx].Volumes()
		if err != nil {
			return "", err
		}
		for jdx := range filesystems {
			if filesystems[jdx].Name() == name {
				return filesystems[jdx].Path(), nil
			}
		}
	}
	return "", fmt.Errorf("Could not find filesystem %v", name)
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *storageModule) ensureCache() error {
	log.Info().Msgf("Setting up cache")

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	// check if we already have a cache
	for idx := range s.volumes {
		filesystems, err := s.volumes[idx].Volumes()
		if err != nil {
			return err
		}
		for jdx := range filesystems {
			if filesystems[jdx].Name() == cacheLabel {
				log.Debug().Msgf("Found existing cache at %v", filesystems[jdx].Path())
				cacheFs = filesystems[jdx]
				break
			}
		}
		if cacheFs != nil {
			break
		}
	}

	if cacheFs == nil {
		log.Debug().Msgf("No cache found, try to create new cache")

		fs, err := s.createSubvol(cacheSize, cacheLabel, modules.SSDDevice)
		if err == modules.ErrNotEnoughSpace {
			// No space on SSD (probably no SSD in the node at all), try HDD
			err = nil
			fs, err = s.createSubvol(cacheSize, cacheLabel, modules.HDDDevice)
		}
		if err != nil {
			return err
		}
		cacheFs = fs
	}

	if !filesystem.IsPathMounted(cacheTarget) {
		log.Debug().Msgf("Mounting cache partition in %s", cacheTarget)
		return filesystem.BindMount(cacheFs, cacheTarget)
	}
	log.Debug().Msgf("Cache partition already mounted in %s", cacheTarget)
	return nil
}

// createSubvol creates a subvolume with the given name and limits it to the given size
// if the requested disk type does not have a storage pool available, an error is
// returned
func (s *storageModule) createSubvol(size uint64, name string, poolType modules.DeviceType) (filesystem.Volume, error) {
	var fs filesystem.Volume
	var err error

	// pick an appropriate pool
	for idx := range s.volumes {
		// ignore pools which don't have the right device type
		if s.volumes[idx].Type() != poolType {
			continue
		}

		usage, err := s.volumes[idx].Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			continue
		}

		reserved, err := s.volumes[idx].Reserved()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get size of pool %s", s.volumes[idx].Name())
		}

		log.Debug().
			Uint64("max size", usage.Size).
			Uint64("reserved", reserved).
			Uint64("new size", reserved+size).
			Msgf("usage of pool %s", s.volumes[idx].Name())
		// Make sure adding this filesystem would not bring us over the disk limit
		if reserved+size > usage.Size {
			log.Info().Msgf("Disk does not have enough space left to hold filesystem")
			continue
		}

		fsn, err := s.volumes[idx].AddVolume(name)
		if err != nil {
			log.Error().Msgf("Failed to create new filesystem: %v", err)
			continue
		}
		if err = fsn.Limit(size); err != nil {
			log.Error().Msgf("Failed to set size limit on new filesystem: %v", err)
			continue
		}
		fs = fsn
		break
	}

	if err != nil || fs != nil {
		return fs, err
	}

	// no error but also no volume, so we didn't manage to create one
	return nil, modules.ErrNotEnoughSpace
}
