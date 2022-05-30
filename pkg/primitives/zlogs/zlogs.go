package zlogs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	var (
		vm      = stubs.NewVMModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
	)

	var cfg zos.ZLogs
	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return nil, errors.Wrap(err, "failed to decode zlogs config")
	}

	machine, err := provision.GetWorkload(ctx, cfg.ZMachine)
	if err != nil || machine.Type != zos.ZMachineType {
		return nil, errors.Wrapf(err, "no zmachine with name '%s'", cfg.ZMachine)
	}

	if !machine.Result.State.IsOkay() {
		return nil, errors.Wrapf(err, "machine state is not ok")
	}

	var machineCfg zos.ZMachine
	if err := json.Unmarshal(machine.Data, &machineCfg); err != nil {
		return nil, errors.Wrap(err, "failed to decode zlogs config")
	}

	var net gridtypes.Name

	if len(machineCfg.Network.Interfaces) > 0 {
		net = machineCfg.Network.Interfaces[0].Network
	} else {
		return nil, fmt.Errorf("invalid zmachine network configuration")
	}

	twin, _ := provision.GetDeploymentID(ctx)

	return nil, vm.StreamCreate(ctx, machine.ID.String(), pkg.Stream{
		ID:        wl.ID.String(),
		Namespace: network.Namespace(ctx, zos.NetworkID(twin, net)),
		Output:    cfg.Output,
	})

}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		vm = stubs.NewVMModuleStub(p.zbus)
	)

	return vm.StreamDelete(ctx, wl.ID.String())
}
