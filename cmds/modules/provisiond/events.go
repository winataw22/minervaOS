package provisiond

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type ContractEventHandler struct {
	node           uint32
	pool           substrate.Manager
	engine         provision.Engine
	eventsConsumer *events.RedisConsumer
}

func NewContractEventHandler(node uint32, mgr substrate.Manager, engine provision.Engine, events *events.RedisConsumer) ContractEventHandler {
	return ContractEventHandler{node: node, pool: mgr, engine: engine, eventsConsumer: events}
}

func (r *ContractEventHandler) current() (map[uint64]gridtypes.Deployment, error) {
	// we need to build a list of all supposedly active contracts on this node
	storage := r.engine.Storage()
	_, deployments, err := storage.Capacity()
	if err != nil {
		return nil, err
	}

	running := make(map[uint64]gridtypes.Deployment)
	for _, active := range deployments {
		running[active.ContractID] = active
	}

	return running, nil
}

func (r *ContractEventHandler) sync(ctx context.Context) error {
	log.Debug().Msg("synchronizing contracts with the chain")

	active, err := r.current()
	if err != nil {
		return errors.Wrap(err, "failed to get current active contracts")
	}
	sub, err := r.pool.Substrate()
	if err != nil {
		return err
	}

	defer sub.Close()
	onchain, err := sub.GetNodeContracts(r.node)
	if err != nil {
		return errors.Wrap(err, "failed to get active node contracts")
	}

	// running will eventually contain all running contracts
	// that also exist on the chain.
	running := make(map[uint64]gridtypes.Deployment)

	for _, contract := range onchain {
		// if contract is in active map, move it to running
		if dl, ok := active[uint64(contract)]; ok {
			running[dl.ContractID] = dl
		}
		// clean the active contracts anyway
		delete(active, uint64(contract))
	}

	// the active map now contains all contracts that are active on the node
	// but not active on the chain (don't exist on chain anymore)
	// hence we can simply deprovision
	for contract, dl := range active {
		// those contracts exits on node but not on chain.
		// need to be deleted
		if err := r.engine.Deprovision(ctx, dl.TwinID, contract, "contract not active on chain"); err != nil {
			log.Error().Err(err).
				Uint32("twin", dl.TwinID).
				Uint64("contract", contract).
				Msg("failed to decomission contract")
		}
	}

	// the running map now contains all contracts that are still exist on the chain.
	// but some of those running contracts can be either locked, or unlocked.
	// hence we need to make sure deployments here has the same state as their contracts

	for id, dl := range running {
		logger := log.With().
			Uint32("twin", dl.TwinID).
			Uint64("contract", id).
			Logger()

		contract, err := sub.GetContract(id)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get contract from chain")
			continue
		}
		// locked is chain state for that contract
		locked := contract.State.IsGracePeriod
		logger.Info().Bool("paused", locked).Msg("contract pause state")
		if locked == r.isLocked(&dl) {
			continue
		}

		// state is different
		logger.Info().Bool("paused", locked).Msg("changing contract pause state")
		action := r.engine.Resume
		if locked {
			action = r.engine.Pause
		}

		if err := action(ctx, dl.TwinID, dl.ContractID); err != nil {
			log.Error().Err(err).Msg("failed to change contract state")
		}
	}

	log.Debug().Msg("synchronization complete")
	return nil
}

func (r *ContractEventHandler) isLocked(dl *gridtypes.Deployment) bool {
	for _, wl := range dl.Workloads {
		if wl.Result.State.IsAny(gridtypes.StatePaused) {
			return true
		}
	}

	return false
}

// Run runs the reporter
func (r *ContractEventHandler) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.
	cancellation, err := r.eventsConsumer.ContractCancelled(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to register to node events")
	}

	locking, err := r.eventsConsumer.ContractLocked(ctx)
	if err != nil {
		return err
	}

	if err := r.sync(ctx); err != nil {
		return errors.Wrap(err, "failed to synchronize active contracts")
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			go func() {
				if err := r.sync(ctx); err != nil {
					log.Error().Err(err).Msg("failed to synchronize contracts with the chain")
				}
			}()
		case event := <-cancellation:
			log.Debug().Msgf("received a cancel contract event %+v", event)

			// otherwise we know what contract to be deleted
			if err := r.engine.Deprovision(ctx, event.TwinId, event.Contract, "contract canceled event received"); err != nil {
				log.Error().Err(err).
					Uint32("twin", event.TwinId).
					Uint64("contract", event.Contract).
					Msg("failed to decomission contract")
			}
		case event := <-locking:
			// todo. we might need to try to sync all contracts to real state if we
			// missed events. so another way to sync here.
			action := r.engine.Resume
			if event.Lock {
				action = r.engine.Pause
			}

			if err := action(ctx, event.TwinId, event.Contract); err != nil {
				log.Error().Err(err).
					Uint32("twin", event.TwinId).
					Uint64("contract", event.Contract).
					Bool("lock", event.Lock).
					Msg("failed to set deployment locking contract")
			}
		}
	}
}
