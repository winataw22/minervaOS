package directory

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database) error {
	if err := directory.Setup(context.TODO(), db); err != nil {
		return err
	}

	var farmAPI FarmAPI
	farms := parent.PathPrefix("/farms").Subrouter()

	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.registerFarm)).Methods("POST")
	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.listFarm)).Methods("GET")
	farms.HandleFunc("/{farm_id}", mw.AsHandlerFunc(farmAPI.getFarm)).Methods("GET")
	// compatibility with gedis_http
	farms.HandleFunc("/list", mw.AsHandlerFunc(farmAPI.cockpitListFarm)).Methods("POST")

	var nodeAPI NodeAPI
	nodes := parent.PathPrefix("/nodes").Subrouter()
	nodes.HandleFunc("", mw.AsHandlerFunc(nodeAPI.registerNode)).Methods("POST")
	nodes.HandleFunc("", mw.AsHandlerFunc(nodeAPI.listNodes)).Methods("GET")
	nodes.HandleFunc("/{node_id}", mw.AsHandlerFunc(nodeAPI.nodeDetail)).Methods("GET")
	nodes.HandleFunc("/{node_id}/interfaces", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerIfaces))).Methods("POST")
	nodes.HandleFunc("/{node_id}/ports", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerPorts))).Methods("POST")
	nodes.HandleFunc("/{node_id}/configure_public", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.configurePublic))).Methods("POST")
	nodes.HandleFunc("/{node_id}/capacity", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerCapacity))).Methods("POST")
	nodes.HandleFunc("/{node_id}/uptime", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateUptimeHandler))).Methods("POST")
	nodes.HandleFunc("/{node_id}/used_resources", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateReservedResources))).Methods("POST")

	// compatibility with gedis_http
	nodes.HandleFunc("/list", mw.AsHandlerFunc(nodeAPI.cockpitListNodes)).Methods("POST")

	return nil
}
