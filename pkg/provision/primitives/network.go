package primitives

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// networkProvision is entry point to provision a network
func (p *Provisioner) networkProvisionImpl(ctx context.Context, reservation *provision.Reservation) error {
	network := &pkg.Network{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return errors.Wrap(err, "failed to unmarshal network from reservation")
	}

	network.NetID = networkID(reservation.User, network.Name)

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", network)).Msg("provision network")

	_, err := mgr.CreateNR(*network)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", network.NetID)
	}

	return nil
}

func (p *Provisioner) networkProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, reservation)
}

func (p *Provisioner) networkDecommission(ctx context.Context, reservation *provision.Reservation) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

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
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}
