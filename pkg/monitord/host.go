package monitord

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/upgrade"
)

// HostMonitor monitor host information
type hostMonitor struct {
	duration time.Duration
}

// NewHostMonitor initialize a new host watcher
func NewHostMonitor(duration time.Duration) (pkg.HostMonitor, error) {
	if duration == 0 {
		duration = 2 * time.Second
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize fs watcher")
	}

	// the file can not exist if the system was booted from overlay
	if _, err := os.Stat(upgrade.FlistInfoFile); err == nil {
		if err := watcher.Add(upgrade.FlistInfoFile); err != nil {
			return nil, errors.Wrapf(err, "failed to watch '%s'", upgrade.FlistInfoFile)
		}
	}

	return &hostMonitor{
		duration: duration,
	}, nil
}

func (h *hostMonitor) Uptime(ctx context.Context) <-chan time.Duration {
	ch := make(chan time.Duration)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(h.duration):
				data, err := os.ReadFile("/proc/uptime")
				if err != nil {
					log.Error().Err(err).Msg("failed to read data from /proc/uptime")
					continue
				}
				var uptime float64
				if _, err := fmt.Sscanf(string(data), "%f", &uptime); err != nil {
					log.Error().Err(err).Msg("failed to parse uptime data")
					continue
				}

				ch <- time.Duration(uptime * float64(time.Second))
			}
		}
	}()

	return ch
}
