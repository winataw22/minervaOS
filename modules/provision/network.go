package provision

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"

	"github.com/threefoldtech/zosv2/modules/stubs"
)

// networkProvision is entry point to provision a network
func networkProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	network := &modules.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal network from reservation")
	}

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))
	log.Debug().Str("network", fmt.Sprintf("%+v", network)).Msg("provision network")

	namespace, err := mgr.ApplyNetResource(*network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create network resource for network %s", network.NetID)
	}
	return namespace, nil
}

func networkDecommission(ctx context.Context, reservation *Reservation) error {
	mgr := stubs.NewNetworkerStub(GetZBus(ctx))

	network := &modules.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return errors.Wrap(err, "failed to unmarshal network from reservation")
	}

	if err := mgr.DeleteNetResource(*network); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}
	return nil
}
