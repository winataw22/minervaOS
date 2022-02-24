package noded

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/go-rmb"
	"github.com/threefoldtech/substrate-client"
)

func runMsgBus(ctx context.Context, sub substrate.Manager, identity substrate.Identity) error {
	// todo: make it argument or parse from broker
	const redis = "/var/run/redis.sock"
	app, err := rmb.NewServer(sub, redis, 100, identity)
	if err != nil {
		return err
	}

	if err := app.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
