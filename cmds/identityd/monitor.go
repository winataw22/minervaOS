package main

import (
	"context"
	"time"

	"github.com/blang/semver"
	"github.com/threefoldtech/zos/pkg"
)

type monitorStream struct {
	duration time.Duration
	version  semver.Version
}

var _ pkg.VersionMonitor = (*monitorStream)(nil)

// newVersionMonitor creates a new instance of version monitor
func newVersionMonitor(d time.Duration, version semver.Version) *monitorStream {
	return &monitorStream{
		duration: d,
	}
}

func (m *monitorStream) GetVersion() semver.Version {
	return m.version
}

func (m *monitorStream) Version(ctx context.Context) <-chan semver.Version {
	ch := make(chan semver.Version)
	go func() {
		defer close(ch)
		ch <- m.version

		for {
			ch <- m.version
			time.Sleep(m.duration)
		}
	}()

	return ch
}
