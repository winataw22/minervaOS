package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
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
	ports, err := stubs.NewNetworkerStub(n.cl).WireguardPorts()
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}

func (n *Network) getPublicConfig(request *http.Request) (interface{}, mw.Response) {
	cfg, err := stubs.NewNetworkerStub(n.cl).GetPublicConfig()
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

	err := stubs.NewNetworkerStub(n.cl).SetPublicConfig(cfg)
	if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}
