package capacityd

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/farmer"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func registration(ctx context.Context, cl zbus.Client) error {
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get runtime environment for zos")
	}

	mgr := stubs.NewIdentityManagerStub(cl)
	storage := stubs.NewStorageModuleStub(cl)

	loc, err := geoip.Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("fetch location")
	}
	oracle := capacity.NewResourceOracle(storage)
	cap, err := oracle.Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	log.Debug().
		Uint64("cru", cap.CRU).
		Uint64("mru", uint64(cap.MRU)).
		Uint64("sru", uint64(cap.SRU)).
		Uint64("hru", uint64(cap.HRU)).
		Msg("node capacity")

	fm, err := env.FarmerClient()
	if err != nil {
		return errors.Wrap(err, "failed to create farmer client")
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		return registerNode(ctx, env, mgr, fm, cap, loc)
	}, bo, retryNotify)

	if err != nil {
		return errors.Wrap(err, "failed to register node")
	}

	log.Info().Msg("node has been registered")
	return nil
}

func retryNotify(err error, d time.Duration) {
	log.Warn().Err(err).Str("sleep", d.String()).Msg("registration failed")
}

func registerNode(ctx context.Context, env environment.Environment, mgr *stubs.IdentityManagerStub, cl *farmer.Client, cap gridtypes.Capacity, loc geoip.Location) error {
	log.Info().Str("id", mgr.NodeID(ctx).Identity()).Msg("start registration of the node")
	log.Info().Msg("registering at farmer bot")

	return cl.NodeRegister(farmer.Node{
		ID:     mgr.NodeID(ctx).Identity(),
		FarmID: uint32(env.FarmerID),
		Secret: env.FarmSecret,
		Location: farmer.Location{

			Longitude: loc.Longitute,
			Latitude:  loc.Latitude,
		},
		Capacity: cap,
	})
}
