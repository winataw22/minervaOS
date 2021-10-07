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

const (
	zstorSocket = "/var/run/zstor.sock"
)

func (p *Primitives) qsfsProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	var result zos.QuatumSafeFSResult
	var proxy zos.QuatumSafeFS
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal qsfs data from reservation: %w", err)
	}
	proxy.Config.Socket = zstorSocket
	qsfs := stubs.NewQSFSDStub(p.zbus)
	path, err := qsfs.Mount(ctx, wl.ID.String(), proxy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup create qsfs mount")
	}
	result.Path = path
	return result, nil
}

func (p *Primitives) qsfsDecommision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	qsfs := stubs.NewQSFSDStub(p.zbus)
	err := qsfs.Unmount(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrap(err, "failed to setup delete qsfs")
	}
	return nil
}
