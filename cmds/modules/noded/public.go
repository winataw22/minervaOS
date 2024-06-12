package noded

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func setPublicConfig(ctx context.Context, cl zbus.Client, cfg *substrate.PublicConfig) error {
	log.Info().Msg("setting node public config")
	netMgr := stubs.NewNetworkerStub(cl)

	if cfg == nil {
		return netMgr.UnsetPublicConfig(ctx)
	}

	pub, err := pkg.PublicConfigFrom(*cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create public config from setup")
	}

	return netMgr.SetPublicConfig(ctx, pub)
}

// public sets and watches changes to public config on chain and tries to apply the provided setup
func public(ctx context.Context, nodeID uint32, cl zbus.Client, events *events.RedisConsumer) error {
	ch, err := events.PublicConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to node events")
	}

	substrateGateway := stubs.NewSubstrateGatewayStub(cl)

reapply:
	for {
		node, err := substrateGateway.GetNode(ctx, nodeID)
		if err != nil {
			return errors.Wrap(err, "failed to get node public config")
		}

		var cfg *substrate.PublicConfig
		if node.PublicConfig.HasValue {
			cfg = &node.PublicConfig.AsValue
		}

		if err := setPublicConfig(ctx, cl, cfg); err != nil {
			return errors.Wrap(err, "failed to set public config (reapply)")
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-ch:
				log.Info().Msgf("got a public config update: %+v", event.PublicConfig)
				var cfg *substrate.PublicConfig
				if event.PublicConfig.HasValue {
					cfg = &event.PublicConfig.AsValue
				}
				if err := setPublicConfig(ctx, cl, cfg); err != nil {
					return errors.Wrap(err, "failed to set public config")
				}
			case <-time.After(2 * time.Hour):
				// last resort, if none of the events
				// was received, it will be a good idea to just
				// check every 2 hours for changes.
				continue reapply
			}
		}
	}
}
