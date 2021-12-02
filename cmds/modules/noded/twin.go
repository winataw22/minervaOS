package noded

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/go-rmb"
)

func runMsgBus(ctx context.Context, twin uint32, substrate string) error {
	// todo: make it argument or parse from broker
	const redis = "/var/run/redis.sock"
	app, err := rmb.NewServer(substrate, redis, int(twin), 100)
	if err != nil {
		return err
	}

	log.Info().Uint32("twin", twin).Msg("starting twin")

	if err := app.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
