package events

import (
	"context"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
)

type Manager struct {
	sub           *substrate.Substrate
	node          uint32
	pubCfg        chan pkg.PublicConfigEvent
	contactCancel chan pkg.ContractCancelledEvent

	o sync.Once
}

func New(sub *substrate.Substrate, node uint32) pkg.Events {
	return &Manager{
		sub:           sub,
		node:          node,
		pubCfg:        make(chan pkg.PublicConfigEvent),
		contactCancel: make(chan pkg.ContractCancelledEvent),
	}
}

func (m *Manager) start(ctx context.Context) {
	log.Info().Msg("start listening to chain events")
	for {
		if err := m.listen(ctx); err != nil {
			log.Error().Err(err).Msg("listening to events failed, retry in 10 seconds")
			<-time.After(10 * time.Second)
		}
	}
}

// Start subscribing and producing events
func (m *Manager) listen(ctx context.Context) error {
reconnect:
	for {
		subClient, meta, err := m.sub.GetClient()
		if err != nil {
			return errors.Wrap(err, "failed to get client to substrate")
		}

		m.pubCfg <- pkg.PublicConfigEvent{Kind: pkg.EventSubscribed}
		m.contactCancel <- pkg.ContractCancelledEvent{Kind: pkg.EventSubscribed}

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
				//m.sub.GetBlock(block types.Hash)
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
						if e.Node != types.U32(m.node) {
							continue
						}
						log.Info().Msgf("got a public config update: %+v", e.Config)
						m.pubCfg <- pkg.PublicConfigEvent{
							Kind:         pkg.EventReceived,
							PublicConfig: e.Config,
						}
					}

					for _, e := range events.SmartContractModule_NodeContractCanceled {
						if e.Node != types.U32(m.node) {
							continue
						}
						log.Info().Uint64("contract", uint64(e.ContractID)).Msg("got contract cancel update")
						m.contactCancel <- pkg.ContractCancelledEvent{
							Kind:     pkg.EventReceived,
							Contract: uint64(e.ContractID),
							TwinId:   uint32(e.Twin),
						}
					}
				}
			case err := <-reg.Err():
				// need a reconnect
				log.Warn().Err(err).Msg("subscription to events stopped, reconnecting")
				continue reconnect
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (m *Manager) PublicConfigEvent(ctx context.Context) <-chan pkg.PublicConfigEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.pubCfg
}

func (m *Manager) ContractCancelledEvent(ctx context.Context) <-chan pkg.ContractCancelledEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.contactCancel
}
