package provisiond

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type ContractEventHandler struct {
	node   uint32
	pool   substrate.Manager
	engine provision.Engine
	cl     zbus.Client
}

func NewContractEventHandler(node uint32, mgr substrate.Manager, engine provision.Engine, cl zbus.Client) ContractEventHandler {
	return ContractEventHandler{node: node, pool: mgr, engine: engine, cl: cl}
}

func (r *ContractEventHandler) current() (map[uint64]uint32, error) {
	// we need to build a list of all supposedly active contracts on this node
	storage := r.engine.Storage()
	twins, err := storage.Twins()
	if err != nil {
		return nil, err
	}
	active := make(map[uint64]uint32)
	for _, twin := range twins {
		deployments, err := storage.ByTwin(twin)
		if err != nil {
			return nil, err
		}
		for _, id := range deployments {
			deployment, err := storage.Get(twin, id)
			if err != nil {
				return nil, err
			}

			if len(deployment.Workloads) > 0 {
				active[deployment.ContractID] = twin
			}
		}
	}

	return active, nil
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

	for _, contract := range onchain {
		delete(active, uint64(contract))
	}

	for contract, twin := range active {
		// those contracts exits on node but not on chain.
		// need to be deleted
		if err := r.engine.Deprovision(ctx, twin, contract, "contract not active on chain"); err != nil {
			log.Error().Err(err).
				Uint32("twin", twin).
				Uint64("contract", contract).
				Msg("failed to decomission contract")
		}
	}

	log.Debug().Msg("synchronization complete")
	return nil
}

// Run runs the reporter
func (r *ContractEventHandler) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.
	events := stubs.NewEventsStub(r.cl)
	cancellation, err := events.ContractCancelledEvent(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to register to node events")
	}

	locking, err := events.ContractLockedEvent(ctx)
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
			if event.Kind == pkg.EventSubscribed {
				// we run this in a go routine because we don't
				// want synchronization of contracts on the chain (that can take some time)
				// to block
				go func() {
					if err := r.sync(ctx); err != nil {
						log.Error().Err(err).Msg("failed to synchronize contracts with the chain")
					}
				}()
				continue
			}

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
