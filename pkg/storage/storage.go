package storage

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/disk"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

const (
	// CacheTarget is the path where the cache disk is mounted
	CacheTarget = "/var/cache"
	cacheLabel  = "zos-cache"
	gib         = 1024 * 1024 * 1024
	cacheSize   = 100 * gib
)

var (
	diskBase = map[pkg.RaidProfile]int{
		pkg.Single: 1,
		pkg.Raid1:  2,
		pkg.Raid10: 4,
	}
)

type storageModule struct {
	volumes       []filesystem.Pool
	brokenPools   []pkg.BrokenPool
	devices       filesystem.DeviceManager
	brokenDevices []pkg.BrokenDevice

	mu sync.RWMutex
}

// New create a new storage module service
func New() (pkg.StorageModule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	m := filesystem.DefaultDeviceManager(ctx)

	m, err := filesystem.Migrate(context.Background(), m)
	if err != nil {
		return nil, err
	}

	s := &storageModule{
		volumes:       []filesystem.Pool{},
		brokenPools:   []pkg.BrokenPool{},
		devices:       m,
		brokenDevices: []pkg.BrokenDevice{},
	}

	// go for a simple linear setup right now
	err = s.initialize(pkg.StoragePolicy{
		Raid:     pkg.Single,
		Disks:    1,
		MaxPools: 0,
	})

	if err == nil {
		log.Info().Msgf("Finished initializing storage module")
	}

	if err := s.Maintenance(); err != nil {
		log.Error().Err(err).Msg("storage devices maintenance failed")
	}

	return s, err
}

// Total gives the total amount of storage available for a device type
func (s *storageModule) Total(kind pkg.DeviceType) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total uint64

	for idx := range s.volumes {
		// ignore pools which don't have the right device type
		if s.volumes[idx].Type() != kind {
			continue
		}

		usage, err := s.volumes[idx].Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			return 0, err
		}

		total += usage.Size
	}
	return total, nil
}

// BrokenPools lists the broken storage pools that have been detected
func (s *storageModule) BrokenPools() []pkg.BrokenPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenPools
}

// BrokenDevices lists the broken devices that have been detected
func (s *storageModule) BrokenDevices() []pkg.BrokenDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenDevices
}

func (s *storageModule) Dump() {
	log.Debug().Int("volumes", len(s.volumes)).Msg("dumping volumes")

	for i := range s.volumes {
		devices := s.volumes[i].Devices()
		for id := range devices {
			d := devices[id]
			log.Debug().Str("path", d.Path).Str("label", d.Label).Str("type", string(d.DiskType)).Send()
		}
	}

}

/**
initialize, must be called at least onetime each boot.
What Initialize will do is the following:
 - Try to mount prepared pools (if they are not mounted already)
 - Scan free devices, apply the policy.
 - If new pools were created, the pool is going to be mounted automatically
**/
func (s *storageModule) initialize(policy pkg.StoragePolicy) error {
	// lock for the entire initialization method, so other code which relies
	// on this observes this as an atomic operation
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Info().Msgf("Initializing storage module")

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
			// volume is already mounted, skip mounting it again, make sure it is
			// in the list of available pools. Since we are in the initialize method,
			// we can safely assume the pool has not been added before, so no need
			// to check for duplicate entries.
			s.volumes = append(s.volumes, volume)
			continue
		}
		_, err = volume.Mount()
		if err != nil {
			s.brokenPools = append(s.brokenPools, pkg.BrokenPool{Label: volume.Name(), Err: err})
			log.Warn().Msgf("Failed to mount volume %v", volume.Name())
			continue
		}
		log.Debug().Msgf("Mounted volume %s", volume.Name())
		s.volumes = append(s.volumes, volume)
	}

	// list disks
	log.Info().Msgf("Finding free disks")
	disks, err := s.devices.Devices(ctx)
	if err != nil {
		return err
	}

	freeDisks := filesystem.DeviceCache{}

	for idx := range disks {
		if !disks[idx].Used() {
			log.Debug().Msgf("Found free device %s", disks[idx].Path)
			freeDisks = append(freeDisks, disks[idx])
		}
	}

	// sort by read time so faster disks are first
	sort.Sort(filesystem.ByReadTime(freeDisks))

	// dumping current s.volumes list
	s.Dump()

	log.Info().Msgf("Creating new volumes using policy: %s", policy.Raid)

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
		if freeDisks[idx].DiskType == pkg.SSDDevice {
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
			log.Debug().Msgf("Creating new volume: %d", i)
			poolDevices := []*filesystem.Device{}

			for j := 0; j < int(policy.Disks); j++ {
				log.Debug().Msgf("Grabbing device %d: %s for new volume", i*int(policy.Disks)+j, fdisks[idx][i*int(policy.Disks)+j].Path)
				poolDevices = append(poolDevices, &fdisks[idx][i*int(policy.Disks)+j])
			}

			pool, err := fs.Create(ctx, uuid.New().String(), policy.Raid, poolDevices...)
			if err != nil {
				log.Info().Err(err).Msg("create filesystem")

				// Failure to create a filesystem -> disk is dead. It is possible
				// that multiple devices are used to create a single pool, and only
				// one devices is actuall broken. We should probably expand on
				// this once we start to use storagepools spanning multiple disks.
				for _, dev := range poolDevices {
					s.brokenDevices = append(s.brokenDevices, pkg.BrokenDevice{Path: dev.Path, Err: err})
				}
				continue
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
				s.brokenPools = append(s.brokenPools, pkg.BrokenPool{Label: newPools[idx].Name(), Err: err})
				continue
			}
			s.volumes = append(s.volumes, newPools[idx])
		}
	}

	if err := filesystem.Partprobe(ctx); err != nil {
		return err
	}

	return s.ensureCache()
}

func (s *storageModule) Maintenance() error {
	for _, pool := range s.volumes {
		log.Info().
			Str("pool", pool.Name()).
			Msg("start storage pool maintained")
		if err := pool.Maintenance(); err != nil {
			log.Error().
				Err(err).
				Str("pool", pool.Name()).
				Msg("error during maintainace")
			return err
		}
		log.Info().
			Str("pool", pool.Name()).
			Msg("finished storage pool maintained")
	}
	return nil
}

// CreateFilesystem with the given size in a storage pool.
func (s *storageModule) CreateFilesystem(name string, size uint64, poolType pkg.DeviceType) (string, error) {
	log.Info().Msgf("Creating new volume with size %d", size)
	if strings.HasPrefix(name, "zdb") {
		return "", fmt.Errorf("invalid volume name. zdb prefix is reserved")
	}

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

	return "", errors.Wrapf(os.ErrNotExist, "subvolume '%s' not found", name)
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *storageModule) ensureCache() error {
	log.Info().Msgf("Setting up cache")

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	if filesystem.IsMountPoint(CacheTarget) {
		log.Debug().Msgf("Cache partition already mounted in %s", CacheTarget)
		return nil
	}

	// check if cache volume available
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

		log.Debug().Msgf("Trying to create new cache on SSD")
		fs, err := s.createSubvol(cacheSize, cacheLabel, pkg.SSDDevice)

		if err != nil {
			log.Warn().Err(err).Msg("failed to create new cache on SSD")
		} else {
			cacheFs = fs
		}
	}

	if cacheFs == nil {
		log.Debug().Msgf("Trying to create new cache on HDD")
		fs, err := s.createSubvol(cacheSize, cacheLabel, pkg.HDDDevice)

		if err != nil {
			log.Warn().Err(err).Msg("failed to create new cache on HDD")
		} else {
			cacheFs = fs
		}
	}

	if cacheFs == nil {
		log.Warn().Msg("failed to create persisted cache disk. Running on limited cache")

		// set limited cache flag
		if err := app.SetFlag("limited-cache"); err != nil {
			return err
		}

		// when everything failed, mount the Tmpfs
		return syscall.Mount("", "/var/cache", "tmpfs", 0, "size=500M")
	}

	log.Info().Msgf("set cache quota to %d GiB", cacheSize/gib)
	if err := cacheFs.Limit(cacheSize); err != nil {
		log.Error().Err(err).Msg("failed to set cache quota")
	}

	log.Debug().Msgf("Mounting cache partition in %s", CacheTarget)
	return filesystem.BindMount(cacheFs, CacheTarget)
}

// createSubvol creates a subvolume with the given name and limits it to the given size
// if the requested disk type does not have a storage pool available, an error is
// returned
func (s *storageModule) createSubvol(size uint64, name string, poolType pkg.DeviceType) (filesystem.Volume, error) {
	var err error

	type Candidate struct {
		Pool      filesystem.Pool
		Available uint64
	}

	var candidates []Candidate

	// pick an appropriate pool
	for _, pool := range s.volumes {
		// ignore pools which don't have the right device type
		if pool.Type() != poolType {
			continue
		}

		usage, err := pool.Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			continue
		}

		reserved, err := pool.Reserved()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get size of pool %s", pool.Name())
			continue
		}

		log.Debug().
			Uint64("max size", usage.Size).
			Uint64("reserved", reserved).
			Uint64("new size", reserved+size).
			Msgf("usage of pool %s", pool.Name())
		// Make sure adding this filesystem would not bring us over the disk limit
		if reserved+size > usage.Size {
			log.Info().Msgf("Disk does not have enough space left to hold filesystem")
			continue
		}

		candidates = append(candidates, Candidate{
			Pool:      pool,
			Available: usage.Size - (reserved + size), // available after new subvolume
		})

	}

	if len(candidates) == 0 {
		return nil, pkg.ErrNotEnoughSpace{DeviceType: poolType}
	}

	sort.Slice(candidates, func(i, j int) bool {
		// reverse sorting so most available is at beginning
		return candidates[i].Available > candidates[j].Available
	})

	var volume filesystem.Volume
	for _, candidate := range candidates {
		volume, err = candidate.Pool.AddVolume(name)
		if err != nil {
			log.Error().Err(err).Str("pool", candidate.Pool.Name()).Msg("failed to create new filesystem")
			continue
		}
		if err = volume.Limit(size); err != nil {
			candidate.Pool.RemoveVolume(volume.Name()) // try to recover
			log.Error().Err(err).Str("volume", volume.Path()).Msg("failed to set volume size limit")
			continue
		}

		return volume, nil
	}

	return nil, fmt.Errorf("failed to create subvolume, logs might have more information")
}

func (s *storageModule) Monitor(ctx context.Context) <-chan pkg.PoolsStats {
	ch := make(chan pkg.PoolsStats)
	values := make(pkg.PoolsStats)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}

			for _, pool := range s.volumes {
				devices, err := s.devices.ByLabel(ctx, pool.Name())
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get devices for pool")
					continue
				}

				var deviceNames []string
				for _, device := range devices {
					deviceNames = append(deviceNames, device.Path)
				}

				usage, err := disk.UsageWithContext(ctx, pool.Path())
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get pool usage")
					continue
				}
				stats, err := disk.IOCountersWithContext(ctx, deviceNames...)
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get io stats for pool")
					continue
				}

				poolStats := pkg.PoolStats{
					UsageStat: *usage,
					Counters:  stats,
				}

				values[pool.Name()] = poolStats
			}

			select {
			case <-ctx.Done():
				return
			case ch <- values:
			}
		}

	}()

	return ch
}
