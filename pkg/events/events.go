package events

import (
	"context"
	"encoding/binary"
	"os"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
)

type Events struct {
	sub   substrate.Manager
	state string
	last  types.BlockNumber

	node           uint32
	pubCfg         chan pkg.PublicConfigEvent
	contractCancel chan pkg.ContractCancelledEvent
	contractLocked chan pkg.ContractLockedEvent

	o sync.Once
}

var (
	_ pkg.Events = (*Events)(nil)
)

func New(sub substrate.Manager, node uint32, state string) *Events {
	return &Events{
		sub:            sub,
		state:          state,
		node:           node,
		pubCfg:         make(chan pkg.PublicConfigEvent),
		contractCancel: make(chan pkg.ContractCancelledEvent),
		contractLocked: make(chan pkg.ContractLockedEvent),
	}
}

func (e *Events) setLatest(num types.BlockNumber) error {
	e.last = num
	var d [4]byte
	binary.BigEndian.PutUint32(d[:], uint32(num))
	if err := os.WriteFile(e.state, d[:], 0644); err != nil {
		return errors.Wrap(err, "failed to commit last block state")
	}
	return nil
}

func (e *Events) getLatest(cl *gsrpc.SubstrateAPI) (types.BlockNumber, error) {
	// getLatest need to
	if e.last != 0 {
		return e.last, nil
	}

	// last is unknown, use last key file
	data, err := os.ReadFile(e.state)
	if err == nil {
		// get latest from the chain
		return types.BlockNumber(binary.BigEndian.Uint32(data)), nil
	}

	block, err := cl.RPC.Chain.GetBlockLatest()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get last block")
	}
	// set latest to previous block because
	return block.Block.Header.Number - 1, nil

}

func (e *Events) eventsTo(cl *gsrpc.SubstrateAPI, meta *types.Metadata, block types.Header) error {
	//
	last, err := e.getLatest(cl)
	if err != nil {
		return errors.Wrap(err, "failed to get latest processed block")
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return err
	}

	start := last + 1
	end := block.Number

	// NOTE: calling QueryStorage with start/end hashes does not work
	// and always return error 'RPC call is unsafe to be called externally'
	for i := start; i <= end; i++ {
		hash, err := cl.RPC.Chain.GetBlockHash(uint64(i))
		if err != nil {
			return errors.Wrapf(err, "failed to get block hash '%d'", start)
		}

		changes, err := cl.RPC.State.QueryStorageAt([]types.StorageKey{key}, hash)
		if err != nil {
			return err
		}

		e.process(changes, meta)
	}

	return nil
}

func (e *Events) process(changes []types.StorageChangeSet, meta *types.Metadata) {
	for _, set := range changes {
		for _, change := range set.Changes {
			if !change.HasStorageData {
				continue
			}

			var events substrate.EventRecords
			if err := types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, &events); err != nil {
				log.Error().Err(err).Msg("failed to decode events from tfchain")
				continue
			}

			for _, event := range events.TfgridModule_NodePublicConfigStored {
				if event.Node != types.U32(e.node) {
					continue
				}
				log.Info().Msgf("got a public config update: %+v", event.Config)
				e.pubCfg <- pkg.PublicConfigEvent{
					PublicConfig: event.Config,
				}
			}

			for _, event := range events.SmartContractModule_NodeContractCanceled {
				if event.Node != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract cancel update")
				e.contractCancel <- pkg.ContractCancelledEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.Twin),
				}
			}

			for _, event := range events.SmartContractModule_ContractGracePeriodStarted {
				if event.NodeID != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period started")
				e.contractLocked <- pkg.ContractLockedEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.TwinID),
					Lock:     true,
				}
			}

			for _, event := range events.SmartContractModule_ContractGracePeriodEnded {
				if event.NodeID != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period ended")
				e.contractLocked <- pkg.ContractLockedEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.TwinID),
					Lock:     false,
				}
			}
		}
	}
}

func (e *Events) subscribe(ctx context.Context) error {
	cl, meta, err := e.sub.Raw()
	if err != nil {
		return errors.Wrap(err, "failed to connect to chain")
	}

	defer cl.Client.Close()
	sub, err := cl.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to new blocks")
	}

	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			return err
		case block := <-sub.Chan():
			err := e.eventsTo(cl, meta, block)
			if err != nil {
				log.Error().Err(err).Msg("failed to process chain events")
				continue
			}

			if err := e.setLatest(block.Number); err != nil {
				return errors.Wrap(err, "failed to commit last block number")
			}
		}
	}
}

func (e *Events) start(ctx context.Context) {
	for {
		err := e.subscribe(ctx)
		if err != nil {
			<-time.After(10 * time.Second)
			continue
		}
		return
	}
}

func (m *Events) PublicConfigEvent(ctx context.Context) <-chan pkg.PublicConfigEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.pubCfg
}

func (m *Events) ContractCancelledEvent(ctx context.Context) <-chan pkg.ContractCancelledEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.contractCancel
}

func (m *Events) ContractLockedEvent(ctx context.Context) <-chan pkg.ContractLockedEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.contractLocked
}
