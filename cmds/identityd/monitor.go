package main

import (
	"context"
	"github.com/blang/semver"
	"github.com/threefoldtech/zos/pkg"
	"time"
)

type monitorStream struct {
	C        chan semver.Version
	duration time.Duration
	version  semver.Version
}

var _ pkg.VersionMonitor = (*monitorStream)(nil)

// newVersionMonitor creates a new instance of version monitor
func newVersionMonitor(d time.Duration) *monitorStream {
	return &monitorStream{
		C:        make(chan semver.Version),
		duration: d,
	}
}

func (m *monitorStream) Version(ctx context.Context) <-chan semver.Version {
	ch := make(chan semver.Version)
	go func() {
		defer close(ch)
		for {
			select {
			case <-time.After(m.duration):
				ch <- m.version
			case v := <-m.C:
				m.version = v
				ch <- m.version
			}
		}
	}()

	return ch
}
