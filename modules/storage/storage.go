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
func New() modules.StorageModule {
	s := &storageModule{
		volumes: []filesystem.Pool{},
		devices: filesystem.DefaultDeviceManager(),
	}

	// go for a simple linear setup right now
	if err := s.initialize(modules.StoragePolicy{
		Raid:     modules.Single,
		Disks:    1,
		MaxPools: 0,
	}); err != nil {
		panic(err)
	}

	return s
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

	if err := s.devices.Scan(context.Background()); err != nil {
		return (err)
	}

	fs := filesystem.NewBtrfs(s.devices)

	// remount all existing pools
	log.Info().Msgf("Remounting existing volumes")
	log.Debug().Msgf("Searching for existing volumes")
	existingPools, err := fs.List(ctx)
	if err != nil {
		return err
	}

	for _, volume := range existingPools {
		if _, mounted := volume.Mounted(); mounted {
			log.Debug().Msgf("Volume %s already mounted", volume.Name())
			// volume is aready mounted, skip mounting it again
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
	freeDisks := []filesystem.Device{}
	for _, device := range disks {
		if !device.Used() {
			log.Debug().Msgf("Found free device %s", device.Path)
			freeDisks = append(freeDisks, device)
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
	ssds := []filesystem.Device{}
	hdds := []filesystem.Device{}

	for _, d := range freeDisks {
		if d.DiskType == filesystem.SSDDevice {
			ssds = append(ssds, d)
		} else {
			hdds = append(hdds, d)
		}
	}

	createdPools := 0
	for _, fdisks := range [][]filesystem.Device{ssds, hdds} {
		possiblePools := len(fdisks) / int(policy.Disks)
		// only create up to the specified amount of pools
		if policy.MaxPools != 0 && int(policy.MaxPools) < possiblePools-createdPools {
			possiblePools = int(policy.MaxPools)
		}
		log.Debug().Msgf("Creating %d new volumes", possiblePools)

		for i := 0; i < possiblePools; i++ {
			log.Debug().Msgf("Creating new volume %d", i)
			poolDevices := []filesystem.Device{}

			for j := 0; j < int(policy.Disks); j++ {
				log.Debug().Msgf("Grabbing device %d: %s for new volume", i*int(policy.Disks)+j, fdisks[i*int(policy.Disks)+j].Path)
				poolDevices = append(poolDevices, fdisks[i*int(policy.Disks)+j])
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
	for _, pool := range newPools {
		if _, mounted := pool.Mounted(); !mounted {
			log.Debug().Msgf("Mounting volume %s", pool.Name())
			if _, err = pool.Mount(); err != nil {
				return err
			}
		}
	}

	s.volumes = append(s.volumes, newPools...)

	// rescan disks
	if err = s.devices.Scan(ctx); err != nil {
		return err
	}

	return s.ensureCache()
}

// CreateFilesystem with the given size in a storage pool.
func (s *storageModule) CreateFilesystem(size uint64) (string, error) {
	log.Info().Msgf("Creating new volume with size %d", size)

	fs, err := s.createSubvol(size, uuid.New().String(), false)
	if err != nil {
		return "", err
	}
	return fs.Path(), nil
}

// ReleaseFilesystem at the given path, this will unmount and then delete
// the filesystem. After this call, the caller must not perform any more actions
// on this filesystem
func (s *storageModule) ReleaseFilesystem(path string) error {
	log.Info().Msgf("Deleting volume at %v", path)

	for _, volume := range s.volumes {
		filesystems, err := volume.Volumes()
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			if fs.Path() == path {
				log.Debug().Msgf("Removing filesystem %v in volume %v", fs.Name(), volume.Name())
				return volume.RemoveVolume(fs.Name())
			}
		}
	}

	log.Warn().Msgf("Could not find filesystem %v", path)
	return nil
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *storageModule) ensureCache() error {
	log.Info().Msgf("Setting up cache")

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	// check if we already have a cache
	for _, volume := range s.volumes {
		filesystems, err := volume.Volumes()
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			if fs.Name() == cacheLabel {
				log.Debug().Msgf("Found existing cache at %v", fs.Path())
				cacheFs = fs
				break
			}
		}
		if cacheFs != nil {
			break
		}
	}

	if cacheFs == nil {
		log.Debug().Msgf("No cache found, try to create new cache")

		fs, err := s.createSubvol(cacheSize, cacheLabel, true)
		if err != nil {
			return err
		}
		cacheFs = fs
	}

	log.Debug().Msgf("Mounting cache partition in %s", cacheTarget)
	return filesystem.BindMount(cacheFs, cacheTarget)
}

// createSubvol creates a subvolume with the given name and limits it to the given size
// if the requested disk type does not have a storage pool available, an error is
// returned
func (s *storageModule) createSubvol(size uint64, name string, preferSSD bool) (filesystem.Volume, error) {
	var fs filesystem.Volume
	var err error

	// sort possible types
	possiblePools := s.volumes
	if preferSSD {
		ssdPools := []filesystem.Pool{}
		hddPools := []filesystem.Pool{}
		for _, pool := range s.volumes {

			if pool.Type() == filesystem.SSDDevice {
				ssdPools = append(ssdPools, pool)
			} else {
				hddPools = append(hddPools, pool)
			}
		}
		possiblePools = append(ssdPools, hddPools...)
	}

	// for now take the first volume, if an ssd is prefered ssds will be sorted
	// first
	for _, volume := range possiblePools {
		usage, err := volume.Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			continue
		}

		// Make sure adding this filesystem would not bring us over the disk limit
		if usage.Used+size > usage.Size {
			log.Info().Msgf("Disk does not have enough space left to hold filesytem")
			continue
		}

		fsn, err := volume.AddVolume(name)
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

	return fs, err
}
