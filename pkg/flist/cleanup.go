package flist

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
)

// Cleaner interface, implementer of this interface
// can start a cleaner job
type Cleaner interface {
	// MountsCleaner runs the clean process, MountsCleaner should be
	// blocking. Caller then can do `go MountsCleaner()` to run it in the background
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
