package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
)

var (
	errStreamDisconnected = fmt.Errorf("stream disconnected")
)

type Manager struct {
	mgr            substrate.Manager
	node           uint32
	pubCfg         chan pkg.PublicConfigEvent
	contractCancel chan pkg.ContractCancelledEvent
	contractLocked chan pkg.ContractLockedEvent

	o sync.Once
}

func New(mgr substrate.Manager, node uint32) pkg.Events {
	return &Manager{
		mgr:            mgr,
		node:           node,
		pubCfg:         make(chan pkg.PublicConfigEvent),
		contractCancel: make(chan pkg.ContractCancelledEvent),
		contractLocked: make(chan pkg.ContractLockedEvent),
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

func (m *Manager) events(ctx context.Context) error {
	subClient, meta, err := m.mgr.Raw()
	if err != nil {
		return errors.Wrap(err, "failed to get client to substrate")
	}

	defer subClient.Client.Close()

	m.pubCfg <- pkg.PublicConfigEvent{Kind: pkg.EventSubscribed}
	m.contractCancel <- pkg.ContractCancelledEvent{Kind: pkg.EventSubscribed}

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
					m.contractCancel <- pkg.ContractCancelledEvent{
						Kind:     pkg.EventReceived,
						Contract: uint64(e.ContractID),
						TwinId:   uint32(e.Twin),
					}
				}

				for _, e := range events.SmartContractModule_ContractGracePeriodStarted {
					if e.NodeID != types.U32(m.node) {
						continue
					}
					log.Info().Uint64("contract", uint64(e.ContractID)).Msg("got contract grace period started")
					m.contractLocked <- pkg.ContractLockedEvent{
						Kind:     pkg.EventReceived,
						Contract: uint64(e.ContractID),
						TwinId:   uint32(e.TwinID),
						Lock:     true,
					}
				}

				for _, e := range events.SmartContractModule_ContractGracePeriodEnded {
					if e.NodeID != types.U32(m.node) {
						continue
					}
					log.Info().Uint64("contract", uint64(e.ContractID)).Msg("got contract grace period ended")
					m.contractLocked <- pkg.ContractLockedEvent{
						Kind:     pkg.EventReceived,
						Contract: uint64(e.ContractID),
						TwinId:   uint32(e.TwinID),
						Lock:     false,
					}
				}
			}
		case err := <-reg.Err():
			// need a reconnect
			return errors.Wrapf(errStreamDisconnected, "subscription to events stopped (%s), reconnecting", err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Start subscribing and producing events
func (m *Manager) listen(ctx context.Context) error {
	for {
		err := m.events(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		} else if errors.Is(err, errStreamDisconnected) {
			continue
		} else {
			return err
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

	return m.contractCancel
}

func (m *Manager) ContractLockedEvent(ctx context.Context) <-chan pkg.ContractLockedEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.contractLocked
}
