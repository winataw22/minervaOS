package noded

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func setPublicConfig(ctx context.Context, cl zbus.Client, cfg substrate.PublicConfig) error {
	log.Info().Msg("setting node public config")
	netMgr := stubs.NewNetworkerStub(cl)

	pub, err := pkg.PublicConfigFrom(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create public config from setup")
	}

	return netMgr.SetPublicConfig(ctx, pub)
}

// public sets and watches changes to public config on chain and tries to apply the provided setup
func public(ctx context.Context, nodeID uint32, env environment.Environment, cl zbus.Client) error {
	mgr, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	stub := stubs.NewEventsStub(cl)
	events, err := stub.PublicConfigEvent(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to node events")
	}

	getNode := func() (*substrate.Node, error) {
		sub, err := mgr.Substrate()
		if err != nil {
			return nil, err
		}

		defer sub.Close()
		return sub.GetNode(nodeID)
	}

reapply:
	for {
		node, err := getNode()
		if err != nil {
			return errors.Wrap(err, "failed to get node public config")
		}

		if node.PublicConfig.HasValue {
			if err := setPublicConfig(ctx, cl, node.PublicConfig.AsValue); err != nil {
				return errors.Wrap(err, "failed to ")
			}
		}

		for event := range events {
			if event.Kind == pkg.EventSubscribed {
				// the events has re-subscribed, so possible
				// loss of events.
				// then we-reapply
				continue reapply
			}

			log.Info().Msgf("got a public config update: %+v", event.PublicConfig)
			if err := setPublicConfig(ctx, cl, event.PublicConfig); err != nil {
				return errors.Wrap(err, "failed to set public config")
			}
		}
	}
}
