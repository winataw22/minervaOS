package mbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

type idArgs struct {
	ContractID uint64 `json:"contract_id"`
}

func (d *Deployments) createOrUpdate(ctx context.Context, payload []byte, update bool) (interface{}, mw.Response) {
	var deployment gridtypes.Deployment
	if err := json.Unmarshal(payload, &deployment); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := deployment.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	if deployment.TwinID != rmb.GetTwinID(ctx) {
		return nil, mw.UnAuthorized(fmt.Errorf("twin id mismatch (deployment: %d, message: %d)", deployment.TwinID, rmb.GetTwinID(ctx)))
	}

	if err := deployment.Verify(d.engine.Twins()); err != nil {
		return nil, mw.UnAuthorized(err)
	}

	// we need to ge the contract here and make sure
	// we can validate the contract against it.

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	action := d.engine.Provision
	if update {
		action = d.engine.Update
	}

	err := action(ctx, deployment)

	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if errors.Is(err, provision.ErrDeploymentExists) {
		return nil, mw.Conflict(err)
	} else if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(err)
	} else if errors.Is(err, provision.ErrDeploymentUpgradeValidationError) {
		return nil, mw.BadRequest(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (d *Deployments) get(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var args idArgs
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, mw.Error(err)
	}

	deployment, err := d.engine.Storage().Get(rmb.GetTwinID(ctx), args.ContractID)
	if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(fmt.Errorf("deployment not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return deployment, nil
}

func (d *Deployments) changes(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var args idArgs
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, mw.Error(err)
	}

	changes, err := d.engine.Storage().Changes(rmb.GetTwinID(ctx), args.ContractID)
	if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(fmt.Errorf("deployment not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return changes, nil
}
