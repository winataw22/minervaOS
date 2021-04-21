package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

// Workloads is provision engine Workloads
type Workloads struct {
	engine provision.Engine
}

// NewWorkloadsAPI creates a new API instance on given gorilla router
func NewWorkloadsAPI(router *mux.Router, engine provision.Engine) (*Workloads, error) {
	api := &Workloads{engine: engine}

	return api, api.setup(router)
}

//Setup setup routes (v1)
func (a *Workloads) setup(router *mux.Router) error {
	//TODO: this will need more twiking later
	//so the user getter will use the grid db
	//plus some internal in-memory cache

	workloads := router.PathPrefix("/deployment").Subrouter()
	workloads.Use(
		mw.NewAuthMiddleware(a.engine.Twins()),
	)

	workloads.Path("/").HandlerFunc(mw.AsHandlerFunc(a.createOrUpdate)).Methods(http.MethodPost, http.MethodPut).Name("workload-create-or-update")
	workloads.Path("/{twin}/{id}").HandlerFunc(mw.AsHandlerFunc(a.get)).Methods(http.MethodGet).Name("workload-get")
	workloads.Path("/{twin}/{id}").HandlerFunc(mw.AsHandlerFunc(a.delete)).Methods(http.MethodDelete).Name("workload-delete")
	return nil
}
