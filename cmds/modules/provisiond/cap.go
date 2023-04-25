package provisiond

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type DeploymentID struct {
	Twin     uint32
	Contract uint64
}

type CapacitySetter struct {
	id      substrate.Identity
	sub     substrate.Manager
	ch      chan DeploymentID
	storage provision.Storage
}

func NewCapacitySetter(id substrate.Identity, sub substrate.Manager, storage provision.Storage) CapacitySetter {
	return CapacitySetter{
		id:      id,
		sub:     sub,
		storage: storage,
		ch:      make(chan DeploymentID, 215),
	}
}

func (c *CapacitySetter) Callback(twin uint32, contract uint64, delete bool) {
	// we don't set capacity on the grid on deletion
	if delete {
		return
	}

	// we just push it to channel so we can return as soon
	// as possible. The channel should have enough capacity
	// to accept enough active contracts.
	c.ch <- DeploymentID{Twin: twin, Contract: contract}
}

func (c *CapacitySetter) setWithClient(cl *substrate.Substrate, deployments ...gridtypes.Deployment) error {
	caps := make([]substrate.ContractResources, 0, len(deployments))
	for _, deployment := range deployments {
		var total gridtypes.Capacity
		for i := range deployment.Workloads {
			wl := &deployment.Workloads[i]
			if wl.Result.State.IsOkay() {
				cap, err := wl.Capacity()
				if err != nil {
					log.Error().Err(err).Str("workload", wl.Name.String()).
						Msg("failed to compute capacity consumption for workload")
					continue
				}

				total.Add(&cap)
			}
		}
		cap := substrate.ContractResources{
			ContractID: types.U64(deployment.ContractID),
			Used: substrate.Resources{
				HRU: types.U64(total.HRU),
				SRU: types.U64(total.SRU),
				CRU: types.U64(total.CRU),
				MRU: types.U64(total.MRU),
			},
		}

		log.Debug().
			Uint64("contract", deployment.ContractID).
			Uint("sru", uint(cap.Used.SRU)).
			Uint("hru", uint(cap.Used.HRU)).
			Uint("mru", uint(cap.Used.MRU)).
			Uint("cru", uint(cap.Used.CRU)).
			Msg("reporting contract usage")

		caps = append(caps, cap)
	}

	bo := backoff.WithMaxRetries(
		backoff.NewConstantBackOff(6*time.Second),
		4,
	)

	return backoff.RetryNotify(func() error {
		return cl.SetContractConsumption(c.id, caps...)
	}, bo, func(err error, d time.Duration) {
		log.Error().Err(err).Dur("retry-in", d).Msg("failed to set contract consumption")
	})
}

func (c *CapacitySetter) Set(deployment ...gridtypes.Deployment) error {
	if len(deployment) == 0 {
		return nil
	}

	cl, err := c.sub.Substrate()
	if err != nil {
		return err
	}

	defer cl.Close()

	return c.setWithClient(cl, deployment...)
}

func (c *CapacitySetter) Run(ctx context.Context) error {
	for {
		var id DeploymentID
		select {
		case <-ctx.Done():
			return nil
		case id = <-c.ch:
		}

		log := log.With().Uint32("twin", id.Twin).Uint64("contract", id.Contract).Logger()

		deployment, err := c.storage.Get(id.Twin, id.Contract)
		if err != nil {
			log.Error().Err(err).Msg("failed to get deployment")
			continue
		}

		if err := c.Set(deployment); err != nil {
			log.Error().Err(err).Msg("failed to set contract usage")
		}
	}
}
