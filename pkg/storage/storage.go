package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/disk"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/kernel"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

const (
	// SSDOverProvisionFactor over provision factor for SSDs
	SSDOverProvisionFactor = 1
	// CacheTarget is the path where the cache disk is mounted
	CacheTarget = "/var/cache"
	// cacheLabel is the name of the cache
	cacheLabel         = "zos-cache"
	gib                = 1024 * 1024 * 1024
	cacheSize          = 5 * gib
	cacheGrowPercent   = 80
	cacheShrinkPercent = 20
	cacheCheckDuration = 30 * time.Minute
)

var (
	_ pkg.StorageModule = (*Module)(nil)
)

// Module implements functionality for pkg.StorageModule
type Module struct {
	devices filesystem.DeviceManager

	ssds []filesystem.Pool
	hdds []filesystem.Pool

	brokenPools   []pkg.BrokenPool
	brokenDevices []pkg.BrokenDevice

	totalSSD uint64
	totalHDD uint64

	mu sync.RWMutex

	// cache is a cache directory can be used with some files
	cache TypeCache
}

type TypeCache struct {
	base string
}

func (t *TypeCache) Set(name string, typ pkg.DeviceType) error {
	if err := os.WriteFile(filepath.Join(t.base, name), []byte(typ), 0644); err != nil {
		return errors.Wrapf(err, "failed to store device type for '%s'", name)
	}

	return nil
}

func (t *TypeCache) Get(name string) (pkg.DeviceType, bool) {
	data, err := os.ReadFile(filepath.Join(t.base, name))
	if err != nil {
		return "", false
	}

	if len(data) == 0 {
		return "", false
	}

	return pkg.DeviceType(data), true
}

// New create a new storage module service
func New(ctx context.Context) (*Module, error) {
	m := filesystem.DefaultDeviceManager()
	cache, err := cache.VolatileDir("storage", 1*cache.Megabyte)
	if err != nil && !os.IsExist(err) {
		return nil, errors.Wrap(err, "failed to create volatile types cache directory")
	}

	s := &Module{
		ssds:          []filesystem.Pool{},
		devices:       m,
		brokenDevices: []pkg.BrokenDevice{},
		cache:         TypeCache{cache},
	}

	// go for a simple linear setup right now
	return s, s.initialize(ctx)
}

// Total gives the total amount of storage available for a device type
func (s *Module) Total(kind pkg.DeviceType) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch kind {
	case zos.SSDDevice:
		return s.totalSSD, nil
	case zos.HDDDevice:
		return s.totalHDD, nil
	default:
		return 0, fmt.Errorf("kind %+v unknown", kind)
	}
}

// BrokenPools lists the broken storage pools that have been detected
func (s *Module) BrokenPools() []pkg.BrokenPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenPools
}

// BrokenDevices lists the broken devices that have been detected
func (s *Module) BrokenDevices() []pkg.BrokenDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenDevices
}

func (s *Module) dump() {
	log.Debug().Int("volumes", len(s.ssds)).Msg("dumping volumes")

	for _, pool := range s.ssds {
		path, err := pool.Mounted()
		if err == nil {
			log.Debug().Msgf("pool %s is mounted at: %s", pool.Name(), path)
		}
		device := pool.Device()
		log.Debug().Str("path", device.Path).Str("label", pool.Name()).Str("type", string(zos.SSDDevice)).Send()
	}

	for _, pool := range s.hdds {
		path, err := pool.Mounted()
		if err == nil {
			log.Debug().Msgf("pool %s is mounted at: %s", pool.Name(), path)
		}
		device := pool.Device()
		log.Debug().Str("path", device.Path).Str("label", pool.Name()).Str("type", string(zos.HDDDevice)).Send()
	}

}

/*
*
initialize, must be called at least onetime each boot.
What Initialize will do is the following:
  - Try to mount prepared pools (if they are not mounted already)
  - Scan free devices, apply the policy.
  - If new pools were created, the pool is going to be mounted automatically

*
*/
func (s *Module) initialize(ctx context.Context) error {
	// lock for the entire initialization method, so other code which relies
	// on this observes this as an atomic operation
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Info().Msgf("Initializing storage module")

	vm := kernel.GetParams().IsVirtualMachine()
	log.Debug().Bool("is-vm", vm).Msg("debugging virtualization detection")

	// Make sure we finish in 1 minute
	subCtx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	devices, err := s.devices.Devices(subCtx)
	if err != nil {
		return err
	}

	for _, device := range devices {
		log.Debug().Msgf("device: %+v", device)
		pool, err := filesystem.NewBtrfsPool(device)
		if err != nil {
			log.Error().Err(err).Str("device", device.Path).Msg("failed to create pool on device")
			s.brokenDevices = append(s.brokenDevices, pkg.BrokenDevice{Path: device.Path, Err: err})
			continue
		}

		_, err = pool.Mount()
		if err != nil {
			log.Error().Err(err).Str("device", device.Path).Msg("failed to mount pool")
			s.brokenPools = append(s.brokenPools, pkg.BrokenPool{Label: pool.Name(), Err: err})
			continue
		}
		usage, err := pool.Usage()
		if err != nil {
			log.Error().Err(err).Str("pool", pool.Name()).Str("device", device.Path).Msg("failed to get usage of pool")
		}

		typ, ok := s.cache.Get(device.Name())
		if !ok {
			log.Debug().Str("device", device.Path).Msg("detecting device type")
			typ, err = device.Type()
			if err != nil {
				log.Error().Str("device", device.Path).Err(err).Msg("failed to check device type")
				continue
			}

			// for development purposes only
			if vm {
				// force ssd device for vms
				typ = zos.SSDDevice

				if device.Path == "/dev/vdd" || device.Path == "/dev/vde" {
					typ = zos.HDDDevice
				}
			}

			log.Debug().Str("device", device.Path).Str("type", typ.String()).Msg("caching device type")
			if err := s.cache.Set(device.Name(), typ); err != nil {
				log.Error().Str("device", device.Path).Err(err).Msg("failed to cache device type")
			}
		} else {
			log.Debug().Str("device", device.Path).Str("type", typ.String()).Msg("device type loaded from cache")
		}

		switch typ {
		case zos.SSDDevice:
			s.totalSSD += usage.Size
			s.ssds = append(s.ssds, pool)
		case zos.HDDDevice:
			s.totalHDD += usage.Size
			s.hdds = append(s.hdds, pool)
		default:
			log.Error().Str("type", string(typ)).Str("device", device.Path).Msg("unknown device type")
		}
	}

	// clean up hdd disks to make sure only zdb subvolumes exists
	// this code makes sure HDDs only have volumes for zdb. Or none
	for _, hdd := range s.hdds {
		// we know that all pools are mounted already so it's okay
		// to access them direction
		volumes, err := hdd.Volumes()
		if err != nil {
			log.Error().Err(err).Str("pool", hdd.Name()).Msg("failed to list pool volumes")
			continue
		}

		for _, vol := range volumes {
			if vol.Name() != zdbVolume && vol.Name() != cacheLabel {
				// we don't delete zdb volumes, also zos-cache volumes
				// although they should not exist on hdd for protection
				// in case of error or bad detection of device speed.
				if err := hdd.RemoveVolume(vol.Name()); err != nil {
					log.Error().Err(err).
						Str("volume", vol.Name()).
						Str("pool", hdd.Name()).
						Msg("failed to delete non zdb volume from harddisk")
				}
			}
		}
	}

	log.Info().
		Int("ssd-pools", len(s.ssds)).
		Int("hdd-pools", len(s.hdds)).
		Int("broken-pools", len(s.brokenPools)).
		Int("broken-devices", len(s.brokenDevices)).
		Msg("pool creations completed")

	s.dump()

	// just in case
	if err := filesystem.Partprobe(subCtx); err != nil {
		return err
	}

	if err := s.ensureCache(ctx); err != nil {
		log.Error().Err(err).Msg("Error ensuring cache")
		return err
	}

	if err := s.shutdownUnusedPools(vm); err != nil {
		log.Error().Err(err).Msg("Error shutting down unused pools")
	}

	s.periodicallyCheckDiskShutdown(vm)

	return nil
}

func (s *Module) poolUsage(pool filesystem.Pool) (uint64, error) {
	_, err := pool.Mounted()
	if errors.Is(err, filesystem.ErrDeviceNotMounted) {
		return 0, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "failed to check pool mount status")
	}

	usage, err := pool.Usage()
	if err != nil {
		return 0, errors.Wrap(err, "failed to check pool usage")
	}

	return usage.Used, nil
}

func (s *Module) Metrics() ([]pkg.PoolMetrics, error) {
	var metrics []pkg.PoolMetrics

	for i, pools := range [][]filesystem.Pool{s.ssds, s.hdds} {
		// this is just to avoid writing the same loop twice
		typ := zos.SSDDevice
		if i == 1 {
			typ = zos.HDDDevice
		}

		for _, pool := range pools {
			size := pool.Device().Size
			used, err := s.poolUsage(pool)

			if err != nil {
				log.Error().Err(err).Msg("failed to check pool usage")
				continue
			}

			metrics = append(metrics, pkg.PoolMetrics{
				Type: typ,
				Name: pool.Name(),
				Size: gridtypes.Unit(size),
				Used: gridtypes.Unit(used),
			})
		}

	}

	return metrics, nil
}

func (s *Module) shutdownUnusedPools(vm bool) error {
	log.Debug().Msg("shutting down unused disks")
	for _, sets := range [][]filesystem.Pool{s.ssds, s.hdds} {
		for _, pool := range sets {
			if _, err := pool.Mounted(); err == nil {
				volumes, err := pool.Volumes()
				if err != nil {
					log.Error().Err(err).Msgf("Failed to retrieve subvolumes on pool %s", pool.Name())
					return err
				}
				log.Debug().Msgf("Pool %s has: %d subvolumes", pool.Name(), len(volumes))

				if len(volumes) > 0 {
					continue
				}
				err = pool.UnMount()
				if err != nil {
					log.Error().Err(err).Msgf("Failed to unmount volume %s", pool.Name())
					return err
				}
			}

			if !vm {
				// only shutdown on physical machine
				if err := pool.Shutdown(); err != nil {
					log.Error().Err(err).Msgf("Error shutting down pool %s", pool.Name())
				}
			}
		}
	}

	return nil
}

// VolumeUpdate updates filesystem size
func (s *Module) VolumeUpdate(name string, size gridtypes.Unit) error {
	_, volume, _, err := s.path(name)
	if err != nil {
		return err
	}

	if err := volume.Limit(uint64(size)); err != nil {
		return err
	}

	return nil
}

// VolumeCreate with the given size in a storage pool.
func (s *Module) VolumeCreate(name string, size gridtypes.Unit) (pkg.Volume, error) {
	log.Info().Msgf("Creating new volume with size %d", size)
	if strings.HasPrefix(name, "zdb") {
		return pkg.Volume{}, fmt.Errorf("invalid volume name. zdb prefix is reserved")
	}

	volume, err := s.VolumeLookup(name)
	if err == nil {
		return volume, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pkg.Volume{}, err
	}

	// otherwise, create a new volume

	fs, err := s.createSubvolWithQuota(size, name)
	if err != nil {
		return pkg.Volume{}, err
	}

	usage, err := fs.Usage()
	if err != nil {
		return pkg.Volume{}, err
	}

	return pkg.Volume{
		Name: fs.Name(),
		Path: fs.Path(),
		Usage: pkg.Usage{
			Size: gridtypes.Unit(usage.Size),
			Used: gridtypes.Unit(usage.Used),
		},
	}, nil
}

// VolumeDelete with the given name, this will unmount and then delete
// the filesystem. After this call, the caller must not perform any more actions
// on this filesystem
func (s *Module) VolumeDelete(name string) error {
	log.Info().Msgf("Deleting volume %v", name)

	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return err
		}
		for _, vol := range volumes {
			if vol.Name() != name {
				continue
			}
			log.Debug().Msgf("Removing filesystem %v in volume %v", vol.Name(), pool.Name())
			err = pool.RemoveVolume(vol.Name())
			if err != nil {
				log.Err(err).Msgf("Error removing volume %s", vol.Name())
				return err
			}
			// if there is only 1 volume, unmount and shutdown pool
			if len(volumes) == 1 {
				err = pool.UnMount()
				if err != nil {
					log.Err(err).Msgf("Error unmounting pool %s", pool.Name())
					return err
				}
				err = pool.Shutdown()
				if err != nil {
					log.Err(err).Msgf("Error shutting down pool %s", pool.Name())
					return err
				}
			}

			return nil
		}
	}

	log.Warn().Msgf("Could not find filesystem %v", name)
	return nil
}

// VolumeList return all the filesystem managed by storeaged present on the nodes
func (s *Module) VolumeList() ([]pkg.Volume, error) {
	fss := make([]pkg.Volume, 0, 10)

	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return nil, err
		}
		for _, v := range volumes {
			// Do not return "special" volumes here
			// instead the GetCacheFS and GetVdiskFS to access them
			if v.Name() == cacheLabel ||
				v.Name() == vdiskVolumeName {
				continue
			}

			usage, err := v.Usage()
			if err != nil {
				return nil, err
			}

			fss = append(fss, pkg.Volume{
				Name: v.Name(),
				Path: v.Path(),
				Usage: pkg.Usage{
					Size: gridtypes.Unit(usage.Size),
					Used: gridtypes.Unit(usage.Used),
				},
			})
		}
	}

	return fss, nil
}

// VolumeLookup return the path of the mountpoint of the named filesystem
// if no volume with name exists, an empty path and an error is returned
func (s *Module) VolumeLookup(name string) (pkg.Volume, error) {
	_, _, fs, err := s.path(name)
	return fs, err
}

// Path return the path of the mountpoint of the named filesystem
// if no volume with name exists, an empty path and an error is returned
func (s *Module) path(name string) (filesystem.Pool, filesystem.Volume, pkg.Volume, error) {
	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}
		filesystems, err := pool.Volumes()
		if err != nil {
			return nil, nil, pkg.Volume{}, err
		}
		for _, fs := range filesystems {
			if fs.Name() == name {
				usage, err := fs.Usage()
				if err != nil {
					return nil, nil, pkg.Volume{}, err
				}

				return pool, fs, pkg.Volume{
					Name: fs.Name(),
					Path: fs.Path(),
					Usage: pkg.Usage{
						Size: gridtypes.Unit(usage.Size),
						Used: gridtypes.Unit(usage.Used),
					},
				}, nil
			}
		}
	}

	return nil, nil, pkg.Volume{}, errors.Wrapf(os.ErrNotExist, "subvolume '%s' not found", name)
}

// Cache return the special filesystem used by 0-OS to store internal state and flist cache
func (s *Module) Cache() (pkg.Volume, error) {
	return s.VolumeLookup(cacheLabel)
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *Module) ensureCache(ctx context.Context) error {
	log.Info().Msgf("Setting up cache")

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	// check if cache volume available
	for _, pool := range s.ssds {
		log.Debug().Str("pool", pool.Name()).Msg("checking pool for cache volume")
		if _, err := pool.Mounted(); err != nil {
			log.Debug().Str("pool", pool.Name()).Msg("pool is not mounted")
			continue
		}

		filesystems, err := pool.Volumes()
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

		log.Debug().Msgf("Trying to create new cache on SSD")
		fs, err := s.createSubvolWithQuota(cacheSize, cacheLabel)

		if err != nil {
			log.Warn().Err(err).Msg("failed to create new cache on SSD")
		} else {
			cacheFs = fs
		}
	}

	if cacheFs == nil {
		log.Warn().Msg("failed to create persisted cache disk. Running on limited cache")

		// set limited cache flag
		if err := app.SetFlag(app.LimitedCache); err != nil {
			return err
		}

		// when everything failed, mount the Tmpfs
		return syscall.Mount("", "/var/cache", "tmpfs", 0, "size=500M")
	}

	_ = app.DeleteFlag(app.LimitedCache)

	go s.watchCache(ctx, cacheFs)

	if !filesystem.IsMountPoint(CacheTarget) {
		log.Debug().Msgf("Mounting cache partition in %s", CacheTarget)
		return filesystem.BindMount(cacheFs, CacheTarget)
	}

	log.Debug().Msgf("Cache partition already mounted in %s", CacheTarget)
	return nil
}

func (s *Module) checkAndResizeCache(cache filesystem.Volume, sizeMultiplier uint64) error {
	usage, err := cache.Usage()
	if err != nil {
		return errors.Wrap(err, "failed to check cache usage")
	}
	log.Debug().Msgf("cache usage %+v", usage)
	percent := usage.Excl * 100 / usage.Size
	size := usage.Size
	if percent >= cacheGrowPercent {
		size += sizeMultiplier
	} else if percent < cacheShrinkPercent && size > sizeMultiplier {
		// if we go below 20% of the size of the cache
		// we can assume the cache size can set comfortably
		// around double the needed space
		size = usage.Excl * 2
		// then ceiled it to number of cache sizes (multipleOf)
		size = ((size / sizeMultiplier) * sizeMultiplier) + sizeMultiplier
	}

	if size < sizeMultiplier {
		size = sizeMultiplier
	}
	if usage.Size == size {
		return nil
	}

	log.Info().Uint64("size", size).Msg("setting cache size")
	return cache.Limit(size)
}

// watchCache will watch the system cache and increase (or decrease)
// it's size as needed
func (s *Module) watchCache(ctx context.Context, cache filesystem.Volume) {
	for {
		if err := s.checkAndResizeCache(cache, cacheSize); err != nil {
			log.Error().Err(err).Msg("error while checking cache size")
		}
		select {
		case <-time.After(cacheCheckDuration):
		case <-ctx.Done():
			return
		}
	}
}

// createSubvolWithQuota creates a subvolume with the given name and limits it to the given size
// if the requested disk type does not have a storage pool with enough free size available, an error is returned
// this methods does set a quota limit equal to size on the created volume
func (s *Module) createSubvolWithQuota(size gridtypes.Unit, name string) (filesystem.Volume, error) {
	volume, err := s.createSubvol(size, name)
	if err != nil {
		return nil, err
	}

	if err = volume.Limit(uint64(size)); err != nil {
		log.Error().Err(err).Str("volume", volume.Path()).Msg("failed to set volume size limit")
		return nil, err
	}

	return volume, nil
}

// createSubvol creates a subvolume with the given name
// if the requested disk type does not have a storage pool with enough free size available, an error is returned
// this method does not set any quota on the subvolume, for this uses createSubvolWithQuota
func (s *Module) createSubvol(size gridtypes.Unit, name string) (filesystem.Volume, error) {
	var err error

	// Look for candidates in mounted pools first
	candidates, err := s.findCandidates(size)
	if err != nil {
		log.Error().Err(err).Msgf("failed to search candidates on mounted pools")
		return nil, err
	}

	var volume filesystem.Volume
	for _, candidate := range candidates {
		volume, err = candidate.Pool.AddVolume(name)
		if err != nil {
			log.Error().Err(err).Str("pool", candidate.Pool.Name()).Msg("failed to create new filesystem")
			continue
		}

		return volume, nil
	}

	return nil, fmt.Errorf("failed to create subvolume, logs might have more information")
}

type candidate struct {
	Pool      filesystem.Pool
	Available uint64
}

func (s *Module) findCandidates(size gridtypes.Unit) ([]candidate, error) {

	// Look for candidates in mounted pools first
	candidates, err := s.checkForCandidates(size, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search candidate on mounted pools")
	}

	log.Debug().Msgf("found %d candidates in mounted pools", len(candidates))

	// If no candidates or found in mounted pools, we check the unmounted pools and get the first one that fits
	if len(candidates) == 0 {
		log.Debug().Msg("Checking unmounted pools")
		candidates, err = s.checkForCandidates(size, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search candidates on unmounted pools")
		}

		log.Debug().Msgf("found %d candidates in unmounted pools", len(candidates))
	}

	if len(candidates) == 0 {
		return nil, pkg.ErrNotEnoughSpace{DeviceType: zos.SSDDevice}
	}

	return candidates, nil
}

func (s *Module) checkForCandidates(size gridtypes.Unit, mounted bool) ([]candidate, error) {
	var candidates []candidate
	for _, pool := range s.ssds {
		_, err := pool.Mounted()
		poolIsMounted := err == nil
		if mounted != poolIsMounted {
			continue
		}

		log.Debug().Msgf("checking pool %s for space", pool.Name())

		if !poolIsMounted && !mounted {
			log.Debug().Msgf("Mounting pool %s...", pool.Name())
			// if the pool is not mounted, and we are looking for not mounted pools, mount it first
			_, err := pool.Mount()
			if err != nil {
				log.Error().Err(err).Msgf("failed to mount pool %s", pool.Name())
				return nil, err
			}
		}

		usage, err := pool.Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			continue
		}

		log.Debug().
			Uint64("pool size", usage.Size).
			Uint64("used", usage.Used).
			Uint64("new used", usage.Used+uint64(size)).
			Msgf("usage of pool %s", pool.Name())

		// Make sure adding this filesystem would not bring us over the disk limit
		if usage.Used+uint64(size) > usage.Size*SSDOverProvisionFactor {
			log.Info().Msgf("Disk does not have enough space left to hold filesystem")

			if !poolIsMounted && !mounted {
				log.Info().Msgf("Previously unmounted pool shutting down again..")
				err = pool.UnMount()
				if err != nil {
					log.Error().Err(err).Msgf("failed to unmount pool %s", pool.Name())
				}
				err = pool.Shutdown()
				if err != nil {
					log.Error().Err(err).Msgf("failed to shutdown pool %s", pool.Name())
				}
			}

			continue
		}

		candidates = append(candidates, candidate{
			Pool:      pool,
			Available: usage.Size*SSDOverProvisionFactor - (usage.Used + uint64(size)),
		})

		// if we are looking for not mounted pools, break here
		if !mounted {
			return candidates, nil
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		// reverse sorting so most available is at beginning
		return candidates[i].Available > candidates[j].Available
	})

	return candidates, nil
}

// Monitor implements monitor method
func (s *Module) Monitor(ctx context.Context) <-chan pkg.PoolsStats {
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

			for _, pool := range s.ssds {
				if _, err := pool.Mounted(); err != nil {
					continue
				}
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

func (s *Module) periodicallyCheckDiskShutdown(vm bool) {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for {
			<-ticker.C
			s.shutdownDisks(vm)
		}
	}()
}

// shutdownDisks will check the disks power status.
// If a disk is on and it is not mounted then it is not supposed to be on, turn it off
func (s *Module) shutdownDisks(vm bool) {
	for _, set := range [][]filesystem.Pool{s.ssds, s.hdds} {
		for _, pool := range set {
			device := pool.Device()
			log.Debug().Msgf("checking device: %s", device.Path)
			on, err := checkDiskPowerStatus(device.Path)
			if err != nil {
				log.Err(err).Msgf("error occurred while checking disk power status")
				continue
			}

			_, err = pool.Mounted()
			if err == nil || !on {
				continue
			}

			if vm {
				continue
			}

			log.Debug().Msgf("shutting down device %s because it is not mounted and the device is on", device.Path)
			err = pool.Shutdown()
			if err != nil {
				log.Err(err).Msgf("failed to shutdown device %s", device.Path)
				continue
			}
		}
	}
}

func checkDiskPowerStatus(path string) (bool, error) {
	output, err := exec.Command("smartctl", "-i", "-n", "standby", path).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	blocks := strings.Split(string(output), "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		if strings.Contains(block, "ACTIVE") {
			return true, nil
		}
	}

	return false, nil
}
