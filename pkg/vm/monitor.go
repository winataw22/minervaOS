package vm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rotate"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	failuresBeforeDestroy = 4
	monitorEvery          = 10 * time.Second
	logrotateEvery        = 10 * time.Minute
)

var (
	// if the failures marker is set to permanent it means
	// the monitoring will not try to restart this machine
	// when it detects that it is down.
	permanent = struct{}{}

	rotator = rotate.NewRotator(
		rotate.MaxSize(8*rotate.Megabytes),
		rotate.TailSize(4*rotate.Megabytes),
	)
)

func (m *Module) logrotate(ctx context.Context) error {
	log.Debug().Msg("running log rotations for vms")
	running, err := FindAll()
	if err != nil {
		return err
	}

	names := make([]string, 0, len(running))
	for name := range running {
		names = append(names, name)
	}

	return rotator.RotateAll(filepath.Join(m.root, logsDir), names...)
}

// Monitor start vms  monitoring
func (m *Module) Monitor(ctx context.Context) {

	go func() {
		monTicker := time.NewTicker(monitorEvery)
		defer monTicker.Stop()
		logTicker := time.NewTicker(logrotateEvery)
		defer logTicker.Stop()

		for {
			select {
			case <-monTicker.C:
				if err := m.monitor(ctx); err != nil {
					log.Error().Err(err).Msg("failed to run monitoring")
				}
			case <-logTicker.C:
				if err := m.logrotate(ctx); err != nil {
					log.Error().Err(err).Msg("failed to run log rotation")
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
	items, err := os.ReadDir(m.cfg)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	running, err := FindAll()
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.IsDir() {
			continue
		}

		id := item.Name()

		if err := m.monitorID(ctx, running, id); err != nil {
			log.Err(err).Str("id", id).Msg("failed to monitor machine")
		}

		// remove vm from running vms
		delete(running, id)
	}

	// now we have running vms that shouldn't be running
	// because they have no config.
	for id, ps := range running {
		log.Info().Str("id", id).Msg("machine is running but not configured")
		_ = syscall.Kill(ps.Pid, syscall.SIGKILL)
	}

	return nil
}

func (m *Module) monitorID(ctx context.Context, running map[string]Process, id string) error {
	stub := stubs.NewProvisionStub(m.client)
	log := log.With().Str("id", id).Logger()

	if ps, ok := running[id]; ok {
		state, exists, err := stub.GetWorkloadStatus(ctx, id)
		if err != nil {
			return errors.Wrapf(err, "failed to get workload status for vm:%s ", id)
		}
		if !exists || state.IsAny(gridtypes.StateDeleted, gridtypes.StateError) {
			log.Debug().Str("name", id).Msg("deleting running vm with no active workload")
			m.removeConfig(id)
			_ = syscall.Kill(ps.Pid, syscall.SIGKILL)
		}
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
		m.removeConfig(id)
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
		vm, err := MachineFromFile(m.configPath(id))

		if err != nil {
			return err
		}

		if vm.NoKeepAlive {
			// if the permanent marker was not set, and we reach here it's possible that
			// the vmd was restarted, hence the in-memory copy of this flag was gone. Hence
			// we need to set it correctly, and just return
			m.failures.Set(id, permanent, cache.NoExpiration)
			return nil
		}

		log.Debug().Str("name", id).Msg("trying to restart the vm")
		if _, err = vm.Run(ctx, m.socketPath(id), m.logsPath(id)); err != nil {
			reason = m.withLogs(m.logsPath(id), err)
		}
	} else {
		reason = fmt.Errorf("deleting vm due to so many crashes")
	}

	if reason != nil {
		log.Debug().Err(reason).Msg("deleting vm due to restart error")
		m.removeConfig(id)

		if err := stub.DecommissionCached(ctx, id, reason.Error()); err != nil {
			return errors.Wrapf(err, "failed to decommission reservation '%s'", id)
		}
	}

	return nil
}
