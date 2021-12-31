package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (p *Primitives) qsfsProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
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

func (p *Primitives) qsfsDecommision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	qsfs := stubs.NewQSFSDStub(p.zbus)
	err := qsfs.SignalDelete(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrap(err, "failed to delete qsfs")
	}
	return nil
}

func (p *Primitives) qsfsUpdate(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
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
