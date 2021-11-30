package mbus

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
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

func (n *Network) interfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := n.listInterfaces(ctx)
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

func (n *Network) setup(router rmb.Router) {

	// network handlers
	sub := router.Subroute("network")
	sub.WithHandler("list_wg_ports", n.listPortsHandler)
	sub.WithHandler("list_public_ips", n.listPublicIPsHandler)
	sub.WithHandler("public_config_get", n.getPublicConfigHandler)
	sub.WithHandler("interfaces", n.interfacesHandler)
}

func (n *Network) listPorts(ctx context.Context) (interface{}, mw.Response) {
	ports, err := stubs.NewNetworkerStub(n.cl).WireguardPorts(ctx)
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}

func (n *Network) listInterfaces(ctx context.Context) (interface{}, mw.Response) {
	mgr := stubs.NewNetworkerStub(n.cl)
	results := make(map[string][]net.IP)
	type q struct {
		inf    string
		ns     string
		rename string
	}
	for _, i := range []q{{"zos", "", "zos"}, {"nygg6", "ndmz", "ygg"}} {
		ips, _, err := mgr.Addrs(ctx, i.inf, i.ns)
		if err != nil {
			return nil, mw.Error(errors.Wrapf(err, "failed to get ips for '%s' interface", i))
		}

		results[i.rename] = func() []net.IP {
			list := make([]net.IP, 0, len(ips))
			for _, item := range ips {
				ip := net.IP(item)
				list = append(list, ip)
			}

			return list
		}()
	}

	return results, nil
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
			workloads := deployment.ByType(zos.PublicIPv4Type, zos.PublicIPType)

			for _, workload := range workloads {
				if workload.Result.State != gridtypes.StateOk {
					continue
				}

				var result zos.PublicIPResult
				if err := workload.Result.Unmarshal(&result); err != nil {
					return nil, mw.Error(err)
				}

				if result.IP.IP != nil {
					ips = append(ips, result.IP.String())
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
