package noded

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
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
func public(ctx context.Context, nodeID uint32, cl zbus.Client) error {
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get runtime environment for zos")
	}
	sub, err := env.GetSubstrate()
	if err != nil {
		return errors.Wrap(err, "failed to get substrate client")
	}

reconnect:
	for {
		subClient, meta, err := sub.GetClient()
		if err != nil {
			return errors.Wrap(err, "failed to get client to substrate")
		}
		// we should try to get the current value of public config from
		node, err := sub.GetNode(nodeID)
		if err != nil {
			return errors.Wrap(err, "failed to get node public config")
		}

		if node.PublicConfig.HasValue {
			if err := setPublicConfig(ctx, cl, node.PublicConfig.AsValue); err != nil {
				return errors.Wrap(err, "failed to ")
			}
		}
		// Subscribe to system events via storage
		key, err := types.CreateStorageKey(meta, "System", "Events", nil)
		if err != nil {
			return errors.Wrap(err, "failed to get storage key for events")
		}

		reg, err := subClient.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
		if err != nil {
			return errors.Wrap(err, "failed to subscribe to events")
		}

		for {
			select {
			case event := <-reg.Chan():
				for _, change := range event.Changes {
					if !change.HasStorageData {
						continue
					}

					var events substrate.EventRecords
					if err := types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, &events); err != nil {
						log.Error().Err(err).Msg("failed to decode events from tfchain")
						continue
					}

					for _, e := range events.TfgridModule_NodePublicConfigStored {
						if e.Node != types.U32(nodeID) {
							continue
						}
						log.Info().Msgf("got a public config update: %+v", e.Config)
						if err := setPublicConfig(ctx, cl, e.Config); err != nil {
							return errors.Wrap(err, "failed to set public config")
						}
					}
				}
			case err := <-reg.Err():
				// need a reconnect
				log.Error().Err(err).Msg("subscription to events stopped, reconnecting")
				continue reconnect
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
