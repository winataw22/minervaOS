package provision

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"

	"github.com/threefoldtech/zosv2/modules/stubs"
)

func networkProvision(ctx context.Context, network *modules.Network) (string, error) {

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))
	log.Debug().Msgf("network %+v", network)
	namespace, err := mgr.ApplyNetResource(*network)
	if err != nil {
		return "", err
	}

	return namespace, err
}

// NetworkProvision is entry point to provision a network
func NetworkProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	network := &modules.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return nil, err
	}

	_, err := networkProvision(ctx, network)
	return nil, err
}
