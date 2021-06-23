package mbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Network message bus api
type Network struct {
	engine provision.Engine
	cl     zbus.Client
}

// NewNetworkMessagebus creates a new messagebus instance
func NewNetworkMessagebus(router rmb.Router, engine provision.Engine, cl zbus.Client) *Network {

	api := &Network{
		engine: engine,
		cl:     cl,
	}
	api.setup(router)
	return api
}

func (n *Network) listPortsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := n.listPorts(ctx)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (n *Network) listPublicIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := n.listPublicIps()
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (n *Network) getPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := n.getPublicConfig(ctx)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (n *Network) setPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	twinID := rmb.GetTwinID(ctx)
	env := environment.MustGet()
	if uint32(env.FarmerID) == twinID {
		return nil, fmt.Errorf("not authorized")
	}

	data, err := n.setPublicConfig(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (n *Network) setup(router rmb.Router) {

	// network handlers
	sub := router.Subroute("network")
	sub.WithHandler("list_wg_ports", n.listPortsHandler)
	sub.WithHandler("list_public_ips", n.listPublicIPsHandler)
	sub.WithHandler("public_config_get", n.getPublicConfigHandler)
	sub.WithHandler("public_config_set", n.setPublicConfigHandler)
}

func (n *Network) listPorts(ctx context.Context) (interface{}, mw.Response) {
	ports, err := stubs.NewNetworkerStub(n.cl).WireguardPorts(ctx)
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}

func (n *Network) listPublicIps() (interface{}, mw.Response) {
	storage := n.engine.Storage()
	// for efficiency this method should just find out configured public Ips.
	// but currently the only way to do this is by scanning the nft rules
	// a nother less efficient but good for now solution is to scan all
	// reservations and find the ones with public IPs.

	twins, err := storage.Twins()
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to list twins"))
	}
	ips := make([]string, 0)
	for _, twin := range twins {
		deploymentsIDs, err := storage.ByTwin(twin)
		if err != nil {
			return nil, mw.Error(errors.Wrap(err, "failed to list twin deployment"))
		}
		for _, id := range deploymentsIDs {
			deployment, err := storage.Get(twin, id)
			if err != nil {
				return nil, mw.Error(errors.Wrap(err, "failed to load deployment"))
			}
			workloads := deployment.ByType(zos.PublicIPType)

			for _, workload := range workloads {
				if workload.Result.State != gridtypes.StateOk {
					continue
				}

				data, err := workload.WorkloadData()
				if err != nil {
					return nil, mw.Error(err)
				}

				ip, _ := data.(*zos.PublicIP)
				if ip.IP.IP != nil {
					ips = append(ips, ip.IP.String())
				}
			}
		}
	}

	return ips, nil
}

func (n *Network) getPublicConfig(ctx context.Context) (interface{}, mw.Response) {
	cfg, err := stubs.NewNetworkerStub(n.cl).GetPublicConfig(ctx)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return cfg, nil
}

func (n *Network) setPublicConfig(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var cfg pkg.PublicConfig

	if err := json.Unmarshal(payload, &cfg); err != nil {
		return nil, mw.BadRequest(err)
	}

	err := stubs.NewNetworkerStub(n.cl).SetPublicConfig(ctx, cfg)
	if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}
