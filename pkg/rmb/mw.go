package rmb

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
)

// LoggerMiddleware simple logger middleware.
func LoggerMiddleware(ctx context.Context, payload []byte) (context.Context, error) {
	msg := GetMessage(ctx)
	log.Debug().
		Uint32("twin", msg.TwinSrc).
		Str("fn", msg.Command).
		Int("body-size", len(payload)).Msg("call")
	return ctx, nil
}

var (
	_ Middleware = LoggerMiddleware
)

// Authorized middleware allows only admins to make these calls
func Authorized(mgr substrate.Manager, farmID uint32) (Middleware, error) {
	sub, err := mgr.Substrate()
	if err != nil {
		return nil, err
	}
	defer sub.Close()
	farm, err := sub.GetFarm(farmID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get farm")
	}

	farmer, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, payload []byte) (context.Context, error) {
		user := GetTwinID(ctx)
		if user != uint32(farmer.ID) {
			return nil, fmt.Errorf("unauthorized")
		}

		return ctx, nil
	}, nil
}
