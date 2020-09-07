package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// networkProvision is entry point to provision a network
func (p *Provisioner) networkProvisionImpl(ctx context.Context, reservation *provision.Reservation) error {
	nr := pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, &nr); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := nr.Valid(); err != nil {
		return fmt.Errorf("validation of the network resource failed: %w", err)
	}

	nr.NetID = provision.NetworkID(reservation.User, nr.Name)

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", nr)).Msg("provision network")

	_, err := mgr.CreateNR(nr)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", nr.NetID)
	}

	return nil
}

func (p *Provisioner) networkProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, reservation)
}

func (p *Provisioner) networkDecommission(ctx context.Context, reservation *provision.Reservation) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	network := &pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	network.NetID = provision.NetworkID(reservation.User, network.Name)

	if err := mgr.DeleteNR(*network); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}
	return nil
}
