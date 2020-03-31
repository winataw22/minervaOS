package workloads

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	"github.com/threefoldtech/zos/tools/explorer/pkg/escrow"
	"github.com/threefoldtech/zos/tools/explorer/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database, escrow *escrow.Escrow) error {
	if err := types.Setup(context.TODO(), db); err != nil {
		return err
	}

	var api API
	api.escrow = escrow
	reservations := parent.PathPrefix("/reservations").Subrouter()

	reservations.HandleFunc("", mw.AsHandlerFunc(api.create)).Methods(http.MethodPost)
	reservations.HandleFunc("", mw.AsHandlerFunc(api.list)).Methods(http.MethodGet)
	reservations.HandleFunc("/{res_id:\\d+}", mw.AsHandlerFunc(api.get)).Methods(http.MethodGet)
	reservations.HandleFunc("/{res_id:\\d+}/sign/provision", mw.AsHandlerFunc(api.signProvision)).Methods(http.MethodPost)
	reservations.HandleFunc("/{res_id:\\d+}/sign/delete", mw.AsHandlerFunc(api.signDelete)).Methods(http.MethodPost)

	reservations.HandleFunc("/workloads/{node_id}", mw.AsHandlerFunc(api.workloads)).Queries("from", "{from:\\d+}").Methods(http.MethodGet)
	reservations.HandleFunc("/workloads/{gwid:\\d+-\\d+}", mw.AsHandlerFunc(api.workloadGet)).Methods(http.MethodGet)
	reservations.HandleFunc("/workloads/{gwid:\\d+-\\d+}/{node_id}", mw.AsHandlerFunc(api.workloadPutResult)).Methods(http.MethodPut)
	reservations.HandleFunc("/workloads/{gwid:\\d+-\\d+}/{node_id}", mw.AsHandlerFunc(api.workloadPutDeleted)).Methods(http.MethodDelete)

	return nil
}
