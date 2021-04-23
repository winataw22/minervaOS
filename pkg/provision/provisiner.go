package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// DeployFunction simple provision function interface
type DeployFunction func(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error)

// RemoveFunction simple decommission function
type RemoveFunction func(ctx context.Context, wl *gridtypes.WorkloadWithID) error

type mapProvisioner struct {
	provisioners    map[gridtypes.WorkloadType]DeployFunction
	decommissioners map[gridtypes.WorkloadType]RemoveFunction
	updaters        map[gridtypes.WorkloadType]DeployFunction
}

// NewMapProvisioner returns a new instance of a map provisioner
func NewMapProvisioner(p map[gridtypes.WorkloadType]DeployFunction, d map[gridtypes.WorkloadType]RemoveFunction, u map[gridtypes.WorkloadType]DeployFunction) Provisioner {
	return &mapProvisioner{
		provisioners:    p,
		decommissioners: d,
		updaters:        u,
	}
}

// Provision implements provision.Provisioner
func (p *mapProvisioner) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error) {
	handler, ok := p.provisioners[wl.Type]
	if !ok {
		return nil, fmt.Errorf("no provisioner associated with workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	data, err := handler(ctx, wl)
	if errors.Is(err, ErrDidNotChange) {
		return nil, err
	}

	return p.buildResult(wl, data, err)
}

// Decommission implementation for provision.Provisioner
func (p *mapProvisioner) Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	handler, ok := p.decommissioners[wl.Type]
	if !ok {
		return fmt.Errorf("no decomissioner associated with workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	return handler(ctx, wl)
}

// Provision implements provision.Provisioner
func (p *mapProvisioner) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error) {
	handler, ok := p.updaters[wl.Type]
	if !ok {
		return nil, fmt.Errorf("no updater associated with workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	data, err := handler(ctx, wl)
	if errors.Is(err, ErrDidNotChange) {
		return nil, err
	}

	return p.buildResult(wl, data, err)
}

func (p *mapProvisioner) CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool {
	_, ok := p.updaters[typ]
	return ok
}

func (p *mapProvisioner) buildResult(wl *gridtypes.WorkloadWithID, data interface{}, err error) (*gridtypes.Result, error) {
	result := &gridtypes.Result{
		Created: gridtypes.Timestamp(time.Now().Unix()),
	}

	if err != nil {
		result.Error = err.Error()
		result.State = gridtypes.StateError
	} else {
		result.State = gridtypes.StateOk
	}

	br, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}
