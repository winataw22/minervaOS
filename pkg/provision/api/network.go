package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
	"github.com/threefoldtech/zos/pkg/stubs"
)

//Network api
type Network struct {
	engine provision.Engine
	cl     zbus.Client
}

// NewNetworkAPI creates a new NetworkAPI
// TODO: this should not be part of the generic api
// because it is zos specific
// may be move with primitives
func NewNetworkAPI(router *mux.Router, engine provision.Engine, cl zbus.Client) (*Network, error) {
	api := &Network{
		engine: engine,
		cl:     cl,
	}
	return api, api.setup(router)
}

func (n *Network) setup(router *mux.Router) error {
	api := router.PathPrefix("/network").Subrouter()
	// public network api
	api.Path("/wireguard").HandlerFunc(mw.AsHandlerFunc(n.listPorts)).Name("network-wireguard-list")
	api.Path("/publicips").HandlerFunc(mw.AsHandlerFunc(n.listPublicIps)).Name("network-publicips-list")
	config := api.PathPrefix("/config").Subrouter()
	config.Path("/public").Methods(http.MethodGet).HandlerFunc(mw.AsHandlerFunc(n.getPublicConfig)).Name("network-get-public-config")
	// set config calls requires admin
	authorized := config.PathPrefix("/public").Subrouter()

	authorized.Use(
		mw.NewAuthMiddleware(n.engine.Admins()),
	)

	authorized.Path("").Methods(http.MethodPost).HandlerFunc(mw.AsHandlerFunc(n.setPublicConfig)).Name("network-set-public-config")
	return nil
}

func (n *Network) listPorts(request *http.Request) (interface{}, mw.Response) {
	ports, err := stubs.NewNetworkerStub(n.cl).WireguardPorts(request.Context())
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}

func (n *Network) listPublicIps(request *http.Request) (interface{}, mw.Response) {
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

func (n *Network) getPublicConfig(request *http.Request) (interface{}, mw.Response) {
	cfg, err := stubs.NewNetworkerStub(n.cl).GetPublicConfig(request.Context())
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return cfg, nil
}

func (n *Network) setPublicConfig(request *http.Request) (interface{}, mw.Response) {
	var cfg pkg.PublicConfig

	if err := json.NewDecoder(request.Body).Decode(&cfg); err != nil {
		return nil, mw.BadRequest(err)
	}

	err := stubs.NewNetworkerStub(n.cl).SetPublicConfig(request.Context(), cfg)
	if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}
