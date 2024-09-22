package flist

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
)

// Cleaner interface, implementer of this interface
// can start a cleaner job
type Cleaner interface {
	// CacheCleaner runs the clean process, CacheCleaner should be
	// blocking. Caller then can do `go CacheCleaner()` to run it in the background
	CacheCleaner(ctx context.Context, every time.Duration, age time.Duration)
}

var _ Cleaner = (*flistModule)(nil)

func (f *flistModule) CacheCleaner(ctx context.Context, every time.Duration, age time.Duration) {
	log := app.SampledLogger()

	// we need to run it at least one time on
	// entry
	if err := f.cleanCache(time.Now(), age); err != nil {
		log.Error().Err(err).Msg("failed to cleanup cache")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
			log.Debug().Msg("running cache cleaner job")
			if err := f.cleanCache(time.Now(), age); err != nil {
				log.Error().Err(err).Msg("failed to clean cache")
			}
		}
	}
}

func (f *flistModule) cleanCache(now time.Time, age time.Duration) error {
	return filepath.Walk(f.cache, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		sys := info.Sys()

		if sys == nil {
			log.Debug().Str("path", path).Msg("failed to check stat of cached file")
			return nil
		}

		if sys, ok := sys.(*syscall.Stat_t); ok {
			// int64 cast required for arm32 targets
			atime := time.Unix(sys.Atim.Sec, sys.Atim.Nsec)

			if now.Sub(atime) > age {
				if err := os.Remove(path); err != nil {
					log.Error().Err(err).Msg("failed to delete cached file")
				}
			}
		}

		return nil
	})
}

// cleanUnusedMounts need to be called ONLY inside the daemon worker
// calling it in a timer can cause an issue if it decide to clean
// ro mount that are still half through a mount operation.
func (f *flistModule) cleanUnusedMounts() error {
	// we list all mounts maintained by flist daemon
	all, err := f.mounts(withUnderPath(f.root))
	if err != nil {
		return errors.Wrap(err, "failed to list flist mounts")
	}

	roTargets := make(map[int64]mountInfo)
	for _, mount := range all.filter(withParentDir(f.ro)) {
		if mount.FSType != fsTypeG8ufs {
			// this mount type should not be under ro
			// where all mounts are g8ufs.
			continue
		}
		g8ufs := mount.AsG8ufs()
		roTargets[g8ufs.Pid] = mount
	}

	for _, mount := range all.filter(withParentDir(f.mountpoint)) {
		var info g8ufsInfo
		switch mount.FSType {
		case fsTypeG8ufs:
			// this is a bind mount
			info = mount.AsG8ufs()
		case fsTypeOverlay:
			// this is an overly mount, so g8ufs lives as a lower layer
			// we get this from the list of mounts.
			lower := all.filter(withTarget(mount.AsOverlay().LowerDir))
			if len(lower) > 0 {
				info = lower[0].AsG8ufs()
			}
		}

		delete(roTargets, info.Pid)
	}
	if len(roTargets) == 0 {
		log.Debug().Msg("no unused mounts detected")
	}
	// cleaning up remaining un-used mounts
	for pid, mount := range roTargets {
		log.Debug().Int64("source", pid).Msgf("cleaning up mount: %+v", mount)
		if err := f.system.Unmount(mount.Target, 0); err != nil {
			log.Error().Err(err).Str("target", mount.Target).Msg("failed to clean up mount")
			continue
		}

		if err := os.RemoveAll(mount.Target); err != nil {
			log.Error().Err(err).Str("target", mount.Target).Msg("failed to delete mountpoint")
		}
	}

	// clean any folder that is not a mount point.
	entries, err := os.ReadDir(f.mountpoint)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(f.mountpoint, entry.Name())
		if err := f.isMountpoint(path); err == nil {
			continue
		}

		if err := os.Remove(path); err != nil {
			log.Error().Err(err).Msgf("failed to clean mountpoint %s", path)
		}
	}

	return nil
}
