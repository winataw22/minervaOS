package vm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	failuresBeforeDestroy = 4
	monitorEvery          = 10 * time.Second
)

var (
	// if the failures marker is set to permanent it means
	// the monitoring will not try to restart this machine
	// when it detects that it is down.
	permanent = struct{}{}
)

// Monitor start vms  monitoring
func (m *Module) Monitor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-time.After(monitorEvery):
				if err := m.monitor(ctx); err != nil {
					log.Error().Err(err).Msg("failed to run monitoring")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Module) monitor(ctx context.Context) error {
	// this lock works with Run call to avoid
	// monitoring trying to restart a machine that is not running yet.
	m.lock.Lock()
	defer m.lock.Unlock()

	// list all machines available under `{root}/firecracker`
	root := filepath.Join(m.root, "firecracker")
	items, err := ioutil.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	running, err := findAll()
	if err != nil {
		return err
	}

	for _, item := range items {
		if !item.IsDir() {
			continue
		}

		id := item.Name()

		if err := m.monitorID(ctx, running, id); err != nil {
			log.Err(err).Str("id", id).Msg("failed to monitor machine")
		}

	}
	return nil
}

func (m *Module) monitorID(ctx context.Context, running map[string]int, id string) error {
	log := log.With().Str("id", id).Logger()

	if _, ok := running[id]; ok {
		return nil
	}

	// otherwise machine is not running. we need to check if we need to restart
	// it

	marker, ok := m.failures.Get(id)
	if !ok {
		// no previous value. so this is the first failure
		m.failures.Set(id, int(0), cache.DefaultExpiration)
	}

	if marker == permanent {
		// if the marker is permanent. it means that this vm
		// is being deleted or not monitored. we don't need to take any more action here
		// (don't try to restart or delete)
		log.Debug().Msg("permanent delete marker is set")
		return nil
	}

	count, err := m.failures.IncrementInt(id, 1)
	if err != nil {
		// this should never happen because we make sure value
		// is set
		return errors.Wrap(err, "failed to check number of failure for the vm")
	}

	var reason error
	if count < failuresBeforeDestroy {
		log.Debug().Msg("trying to restart the vm")
		jailed, err := JailedFromPath(filepath.Join(m.root, "firecracker", id, "root"))
		if err != nil {
			return err
		}

		if jailed.NoKeepAlive {
			// if the permanent marker was not set, and we reach here it's possible that
			// the vmd was restarted, hence the in-memory copy of this flag was gone. Hence
			// we need to set it correctly, and just return
			m.failures.Set(id, permanent, cache.NoExpiration)
			return nil
		}

		reason = jailed.Start(ctx)
		if reason == nil {
			reason = m.waitAndAdjOom(ctx, id)
		}
	} else {
		reason = fmt.Errorf("deleting vm due to so many crashes")
	}

	if reason != nil {
		log.Debug().Err(reason).Msg("deleting vm due to restart error")

		stub := stubs.NewProvisionStub(m.client)
		if err := stub.DecommissionCached(id, reason.Error()); err != nil {
			if err := m.cleanFs(id); err != nil {
				log.Error().Err(err).Msg("failed to delete clean up unmanaged vm")
			}

			return errors.Wrapf(err, "failed to decommission reservation '%s'", id)
		}
	}

	return nil
}
