package qsfs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*Manager)(nil)
	_ provision.Updater = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	var result zos.QuatumSafeFSResult
	var proxy zos.QuantumSafeFS
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal qsfs data from reservation: %w", err)
	}
	qsfs := stubs.NewQSFSDStub(p.zbus)
	info, err := qsfs.Mount(ctx, wl.ID.String(), proxy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create qsfs mount")
	}
	result.Path = info.Path
	result.MetricsEndpoint = info.MetricsEndpoint
	return result, nil
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	qsfs := stubs.NewQSFSDStub(p.zbus)
	err := qsfs.SignalDelete(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrap(err, "failed to delete qsfs")
	}
	return nil
}

func (p *Manager) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	var result zos.QuatumSafeFSResult
	var proxy zos.QuantumSafeFS
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal qsfs data from reservation: %w", err)
	}
	qsfs := stubs.NewQSFSDStub(p.zbus)
	info, err := qsfs.UpdateMount(ctx, wl.ID.String(), proxy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update qsfs mount")
	}
	result.Path = info.Path
	result.MetricsEndpoint = info.MetricsEndpoint
	return result, nil
}
