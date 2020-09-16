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
)

// Cleaner interface, implementer of this interface
// can start a cleaner job
type Cleaner interface {
	// Cleaner runs the clean process, Cleaner should be
	// blocking. Caller then can do `go Cleaner()` to run it in the background
	Cleaner(ctx context.Context, every time.Duration)
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
func (f *flistModule) cleanupMount(name string) error {
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

	volume, err := f.storage.Path(name)
	if err != nil {
		log.Warn().Err(err).Str("subvolume", name).Msg("subvolume does not exist")
		return nil
	}

	ro := filepath.Join(volume, "ro")
	if err := syscall.Unmount(ro, syscall.MNT_DETACH); err != nil {
		log.Warn().Err(err).Str("path", ro).Msg("fail to unmount ro layer")
	}

	return nil
}

func (f *flistModule) cleanupAll() error {
	mounts, err := f.listMounts()
	if err != nil {
		return errors.Wrap(err, "failed to list current possible mounts")
	}

	for name, pid := range mounts {
		switch pid {
		case -1:
			// process has shutdown gracefully
			log.Debug().Str("name", name).Int64("pid", pid).Msg("attempt to clean up mount")
			f.cleanupMount(name)
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
				f.cleanupMount(name)
			}
			// nothing to do
		}
	}

	return nil
}

// Cleaner runs forever, checks the tracker files for filesystem processes
// that requires cleanup
func (f *flistModule) Cleaner(ctx context.Context, every time.Duration) {
	for {
		select {
		case <-ctx.Done():
		case <-time.After(every):
			log.Debug().Msg("running cleaner job")
			if err := f.cleanupAll(); err != nil {
				log.Error().Err(err).Msg("failed to cleanup stall mounts")
			}
		}
	}
}
