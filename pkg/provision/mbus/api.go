package mbus

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// Deployments message bus API
type Deployments struct {
	engine provision.Engine
}

// NewDeploymentMessageBus creates and register a new deployment api
func NewDeploymentMessageBus(router rmb.Router, engine provision.Engine) *Deployments {

	d := Deployments{
		engine: engine,
	}

	d.setup(router)
	return &d
}

func (d *Deployments) setup(router rmb.Router) {
	sub := router.Subroute("deployment")

	// zos deployment handlers
	sub.WithHandler("deploy", d.deployHandler)
	sub.WithHandler("update", d.updateHandler)
	sub.WithHandler("delete", d.deleteHandler)
	sub.WithHandler("get", d.getHandler)
	sub.WithHandler("changes", d.changesHandler)
}

func (d *Deployments) updateHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := d.createOrUpdate(ctx, payload, true)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (d *Deployments) deployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := d.createOrUpdate(ctx, payload, false)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (d *Deployments) deleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return nil, fmt.Errorf("deletion over the api is disabled, please cancel your contract instead")

	// code disabled.

	// data, err := d.delete(ctx, payload)
	// if err != nil {
	// 	return nil, err.Err()
	// }
	// return data, nil
}

func (d *Deployments) getHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := d.get(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (d *Deployments) changesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := d.changes(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}
