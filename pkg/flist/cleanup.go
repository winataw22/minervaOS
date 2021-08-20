package flist

import (
	"context"
	"fmt"
	"io/ioutil"
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
	// MountsCleaner runs the clean process, MountsCleaner should be
	// blocking. Caller then can do `go MountsCleaner()` to run it in the background
	MountsCleaner(ctx context.Context, every time.Duration)
	CacheCleaner(ctx context.Context, every time.Duration, age time.Duration)
}

var _ Cleaner = (*flistModule)(nil)

func (f *flistModule) listMounts() (map[string]int64, error) {
	infos, err := ioutil.ReadDir(f.run)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list g8ufs process pids: %s", f.pid)
	}

	data := make(map[string]int64)
	for _, info := range infos {
		if info.IsDir() {
			// what is that doing here?
			continue
		}

		path := filepath.Join(f.run, info.Name())
		pid, err := f.getPid(path)
		if os.IsNotExist(err) {
			pid = -1
		} else if err != nil {
			log.Error().Err(err).Str("pid-file", path).Msg("invalid pid file")
		}

		data[info.Name()] = pid
	}

	return data, nil
}

// cleanupMount forces clean up of a mount point
func (f *flistModule) cleanupMount(ctx context.Context, name string) error {
	path, err := f.mountpath(name)
	if err != nil {
		return err
	}

	// always clean up files
	defer func() {
		for _, file := range []string{
			filepath.Join(f.pid, fmt.Sprintf("%s.pid", name)),
			filepath.Join(f.run, name),
			filepath.Join(f.log, fmt.Sprintf("%s.log", name)),
		} {
			if err := os.Remove(file); err != nil {
				log.Warn().Err(err).Str("file", file).Msg("failed to delete pid file")
			}
		}
	}()
	// the following procedure will ignore errors
	// because the only point of this is to force
	// clean the entire mount.
	// so no checks or error checking is done

	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		log.Warn().Err(err).Str("path", path).Msg("fail to unmount flist")
	}

	fs, err := f.storage.VolumeLookup(ctx, name)
	if err != nil {
		log.Warn().Err(err).Str("subvolume", name).Msg("subvolume does not exist")
		return nil
	}

	ro := filepath.Join(fs.Path, "ro")
	if err := syscall.Unmount(ro, syscall.MNT_DETACH); err != nil {
		log.Warn().Err(err).Str("path", ro).Msg("fail to unmount ro layer")
	}

	return nil
}

func (f *flistModule) cleanupAll(ctx context.Context) error {
	mounts, err := f.listMounts()
	if err != nil {
		return errors.Wrap(err, "failed to list current possible mounts")
	}

	for name, pid := range mounts {
		switch pid {
		case -1:
			// process has shutdown gracefully
			log.Debug().Str("name", name).Int64("pid", pid).Msg("attempt to clean up mount")
			if err := f.cleanupMount(ctx, name); err != nil {
				log.Error().Err(err).Str("name", name).Msg("failed to clean up mounts")
			}
		case 0:
			// the file exists, but we can't read the file content
			// for some reason!
			// TODO: ?
		default:
			// valid PID.
			_, err := f.getMountOptionsForPID(pid)
			if err != nil {
				// this is only possible if process does not exist.
				// hence we need to clean up.
				log.Debug().Str("name", name).Int64("pid", pid).Msg("attempt to clean up mount")
				if err := f.cleanupMount(ctx, name); err != nil {
					log.Error().Err(err).Str("name", name).Msg("failed to clean up mounts")
				}
			}
			// nothing to do
		}
	}

	return nil
}

// Cleaner runs forever, checks the tracker files for filesystem processes
// that requires cleanup
func (f *flistModule) MountsCleaner(ctx context.Context, every time.Duration) {
	log := app.SampledLogger()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
			log.Debug().Msg("running cleaner job")
			if err := f.cleanupAll(ctx); err != nil {
				log.Error().Err(err).Msg("failed to cleanup stall mounts")
			}
		}
	}
}

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
			atime := time.Unix(int64(sys.Atim.Sec), int64(sys.Atim.Nsec))

			if now.Sub(atime) > age {
				if err := os.Remove(path); err != nil {
					log.Error().Err(err).Msg("failed to delete cached file")
				}
			}
		}

		return nil
	})
}
