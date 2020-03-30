package directory

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	directory "github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"
	"github.com/zaibon/httpsig"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database) error {
	if err := directory.Setup(context.TODO(), db); err != nil {
		return err
	}

	var farmAPI FarmAPI
	farms := parent.PathPrefix("/farms").Subrouter()
	farmsAuthenticated := parent.PathPrefix("/farms").Subrouter()
	farmsAuthenticated.Use(mw.NewAuthMiddleware(httpsig.NewVerifier(mw.NewUserKeyGetter(db))).Middleware)

	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.registerFarm)).Methods("POST").Name("farm-register")
	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.listFarm)).Methods("GET").Name("farm-list")
	farms.HandleFunc("/{farm_id}", mw.AsHandlerFunc(farmAPI.getFarm)).Methods("GET").Name("farm-get")
	farmsAuthenticated.HandleFunc("/{farm_id}", mw.AsHandlerFunc(farmAPI.updateFarm)).Methods("PUT").Name("farm-update")

	var nodeAPI NodeAPI
	nodes := parent.PathPrefix("/nodes").Subrouter()
	nodesAuthenticated := parent.PathPrefix("/nodes").Subrouter()
	userAuthenticated := parent.PathPrefix("/nodes").Subrouter()

	nodeAuthMW := mw.NewAuthMiddleware(httpsig.NewVerifier(mw.NewNodeKeyGetter()))
	userAuthMW := mw.NewAuthMiddleware(httpsig.NewVerifier(mw.NewUserKeyGetter(db)))

	userAuthenticated.Use(userAuthMW.Middleware)
	nodesAuthenticated.Use(nodeAuthMW.Middleware)

	nodes.HandleFunc("", mw.AsHandlerFunc(nodeAPI.registerNode)).Methods("POST").Name("node-register")
	nodes.HandleFunc("", mw.AsHandlerFunc(nodeAPI.listNodes)).Methods("GET").Name("nodes-list")
	nodes.HandleFunc("/{node_id}", mw.AsHandlerFunc(nodeAPI.nodeDetail)).Methods("GET").Name(("node-get"))
	nodesAuthenticated.HandleFunc("/{node_id}/interfaces", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerIfaces))).Methods("POST").Name("node-interfaces")
	nodesAuthenticated.HandleFunc("/{node_id}/ports", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerPorts))).Methods("POST").Name("node-set-ports")
	userAuthenticated.HandleFunc("/{node_id}/configure_public", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.configurePublic))).Methods("POST").Name("node-configure-public")
	userAuthenticated.HandleFunc("/{node_id}/configure_free", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.configureFreeToUse))).Methods("POST").Name("node-configure-free")
	nodesAuthenticated.HandleFunc("/{node_id}/capacity", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerCapacity))).Methods("POST").Name("node-capacity")
	nodesAuthenticated.HandleFunc("/{node_id}/uptime", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateUptimeHandler))).Methods("POST").Name("node-uptime")
	nodesAuthenticated.HandleFunc("/{node_id}/used_resources", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateReservedResources))).Methods("POST").Name("node-reserved-resources")

	return nil
}
