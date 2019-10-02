package provision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/stubs"
)

// networkProvision is entry point to provision a network
func networkProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	network := &pkg.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal network from reservation")
	}

	network.NetID = networkID(reservation.User, network.Name)

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))
	log.Debug().Str("network", fmt.Sprintf("%+v", network)).Msg("provision network")
	log.Debug().Str("nr", fmt.Sprintf("%+v", network.NetResources[0])).Msg("provision network")

	_, err := mgr.CreateNR(*network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create network resource for network %s", network.NetID)
	}

	// nothing to return to BCDB
	return nil, nil
}

func networkDecommission(ctx context.Context, reservation *Reservation) error {
	mgr := stubs.NewNetworkerStub(GetZBus(ctx))

	network := &pkg.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return errors.Wrap(err, "failed to unmarshal network from reservation")
	}

	network.NetID = networkID(reservation.User, network.Name)

	if err := mgr.DeleteNR(*network); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}
	return nil
}

func networkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	b := base58.Encode(buf.Bytes())[:13]
	return pkg.NetID(string(b))
}
