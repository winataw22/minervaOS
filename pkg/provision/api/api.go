package api

import (
	"net/http"

	"github.com/google/uuid"
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

func (a *Workloads) nextID() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

//Setup setup routes (v1)
func (a *Workloads) setup(router *mux.Router) error {
	//TODO: this will need more twiking later
	//so the user getter will use the grid db
	//plus some internal in-memory cache

	workloads := router.PathPrefix("/workloads").Subrouter()
	workloads.Use(
		mw.NewAuthMiddleware(a.engine.Users()),
	)

	workloads.Path("/").HandlerFunc(mw.AsHandlerFunc(a.create)).Methods(http.MethodPost).Name("workload-create")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.get)).Methods(http.MethodGet).Name("workload-get")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.delete)).Methods(http.MethodDelete).Name("workload-delete")
	return nil
}
