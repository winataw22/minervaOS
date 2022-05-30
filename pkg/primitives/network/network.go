package network

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
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

// networkProvision is entry point to provision a network
func (p *Manager) networkProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	twin, _ := provision.GetDeploymentID(ctx)

	var network zos.Network
	if err := json.Unmarshal(wl.Data, &network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", network)).Msg("provision network")

	_, err := mgr.CreateNR(ctx, pkg.Network{
		Network: network,
		NetID:   zos.NetworkID(twin, wl.Name),
	})

	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", wl.ID)
	}

	return nil
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, wl)
}

func (p *Manager) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, wl)
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	var network zos.Network
	if err := json.Unmarshal(wl.Data, &network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	twin, _ := provision.GetDeploymentID(ctx)

	if err := mgr.DeleteNR(ctx, pkg.Network{
		Network: network,
		NetID:   zos.NetworkID(twin, wl.Name),
	}); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}

	return nil
}
