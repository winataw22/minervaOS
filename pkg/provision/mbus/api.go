package mbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
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
	sub.WithHandler("list", d.listHandler)
	sub.WithHandler("changes", d.changesHandler)

	net := router.Subroute("network")
	net.WithHandler("list_public_ips", d.listPublicIps)
	net.WithHandler("list_private_ips", d.listPrivateIps)
}

func (n *Deployments) listPublicIps(ctx context.Context, _ []byte) (interface{}, error) {
	storage := n.engine.Storage()
	// for efficiency this method should just find out configured public Ips.
	// but currently the only way to do this is by scanning the nft rules
	// a nother less efficient but good for now solution is to scan all
	// reservations and find the ones with public IPs.

	twins, err := storage.Twins()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list twins")
	}
	ips := make([]string, 0)
	for _, twin := range twins {
		deploymentsIDs, err := storage.ByTwin(twin)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list twin deployment")
		}
		for _, id := range deploymentsIDs {
			deployment, err := storage.Get(twin, id)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load deployment")
			}
			workloads := deployment.ByType(zos.PublicIPv4Type, zos.PublicIPType)

			for _, workload := range workloads {
				if workload.Result.State != gridtypes.StateOk {
					continue
				}

				var result zos.PublicIPResult
				if err := workload.Result.Unmarshal(&result); err != nil {
					return nil, err
				}

				if result.IP.IP != nil {
					ips = append(ips, result.IP.String())
				}
			}
		}
	}

	return ips, nil
}

type listPrivateIpsArgs struct {
	NetworkName gridtypes.Name `json:"network_name"`
}

func (d *Deployments) listPrivateIps(ctx context.Context, payload []byte) (interface{}, error) {
	var args listPrivateIpsArgs
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	deployments, err := d.list(ctx, payload)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0)
	for _, deployment := range deployments {
		vms := deployment.ByType(zos.ZMachineType)
		for _, vm := range vms {
			if vm.Result.State.IsAny(gridtypes.StateDeleted, gridtypes.StateError) {
				continue
			}
			data, err := vm.WorkloadData()
			if err != nil {
				return nil, err
			}
			zmachine := data.(*zos.ZMachine)
			for _, inf := range zmachine.Network.Interfaces {
				if inf.Network == args.NetworkName {
					ips = append(ips, inf.IP.String())
				}
			}
		}
	}
	return ips, nil
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

func (d *Deployments) listHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return d.list(ctx, payload)
}
