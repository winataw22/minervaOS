package provisiond

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rusart/muxprom"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/primitives"
	"github.com/threefoldtech/zos/pkg/provision/api"
	"github.com/threefoldtech/zos/pkg/provision/storage"
	"github.com/threefoldtech/zos/pkg/substrate"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	module = "provision"
	gib    = 1024 * 1024 * 1024
)

// Module entry point
var Module cli.Command = cli.Command{
	Name:  "provisiond",
	Usage: "handles reservations streams and use other daemon to deploy them",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/provisiond",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.StringFlag{
			Name:  "http",
			Usage: "http listen address",
			Value: ":2021",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		rootDir      string = cli.String("root")
		httpAddr     string = cli.String("http")
	)

	// keep checking if limited-cache flag is set
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("failed cache reservation! Retrying every 30 seconds...")
		for app.CheckFlag(app.LimitedCache) {
			time.Sleep(time.Second * 30)
		}
	}

	if err := os.MkdirAll(rootDir, 0770); err != nil {
		return errors.Wrap(err, "failed to create cache directory")
	}

	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to parse node environment")
	}

	if env.Orphan {
		// disable providiond on this node
		// we don't have a valid farmer id set
		log.Info().Msg("orphan node, we won't provision anything at all")
		select {}
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "failed to connect to message broker")
	}
	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(cl)
	nodeID := identity.NodeID()

	// block until networkd is ready to serve request from zbus
	// this is used to prevent uptime and online status to the explorer if the node is not in a fully ready
	// https://github.com/threefoldtech/zos/issues/632
	// NOTE - UPDATE: this block of code should be deprecated
	// since we do the waiting in zinit now since provisiond waits for networkd
	// which has a 'test' condition in the zinit yaml file for networkd to wait
	// for zbus
	network := stubs.NewNetworkerStub(cl)
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	backoff.RetryNotify(func() error {
		return network.Ready()
	}, bo, func(err error, d time.Duration) {
		log.Error().Err(err).Msg("networkd is not ready yet")
	})

	router := mux.NewRouter().StrictSlash(true)

	prom := muxprom.New(
		muxprom.Router(router),
		muxprom.Namespace("provision"),
	)
	prom.Instrument()

	// the v1 endpoint will be used by all components to register endpoints
	// that are specific for that component
	v1 := router.PathPrefix("/api/v1").Subrouter()
	// keep track of resource units reserved and amount of workloads provisionned

	// to store reservation locally on the node
	store, err := storage.NewFSStore(filepath.Join(rootDir, "workloads"))
	if err != nil {
		return errors.Wrap(err, "failed to create local reservation store")
	}

	provisioners := primitives.NewPrimitivesProvisioner(cl)

	// update initial capacity with
	reserved, err := getNodeReserved(cl)
	if err != nil {
		return errors.Wrap(err, "failed to get node reserved capacity")
	}
	cap, err := capacity.NewResourceOracle(stubs.NewStorageModuleStub(cl)).Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	// statistics collects information about workload statistics
	// also does some checks on capacity
	statistics := primitives.NewStatistics(
		cap,
		reserved,
		nodeID.Identity(),
		provisioners,
	)

	// add endpoint for statistics
	if err := primitives.NewStatisticsAPI(v1, statistics); err != nil {
		return errors.Wrap(err, "failed to create statistics api")
	}

	// TODO: that is a test user map for development, do not commit
	// users := mw.NewUserMap()
	// users.AddKeyFromHex(1, "95d1ba20e9f5cb6cfc6182fecfa904664fb1953eba520db454d5d5afaa82d791")

	users, err := substrate.NewSubstrateTwins(env.SubstrateURL)
	if err != nil {
		return errors.Wrap(err, "failed to create substrate users database")
	}

	admins, err := substrate.NewSubstrateAdmins(env.SubstrateURL, uint32(env.FarmerID))
	if err != nil {
		return errors.Wrap(err, "failed to create substrate admins database")
	}

	queues := filepath.Join(rootDir, "queues")
	if err := os.MkdirAll(queues, 0755); err != nil {
		return errors.Wrap(err, "failed to create storage for queues")
	}

	engine, err := provision.New(
		store,
		statistics,
		queues,
		provision.WithTwins(users),
		provision.WithAdmins(admins),
		// set priority to some reservation types on boot
		// so we always need to make sure all volumes and networks
		// comes first.
		provision.WithStartupOrder(
			zos.VolumeType,
			zos.NetworkType,
			zos.PublicIPType,
		),
		provision.WithRerunAll(app.IsFirstBoot(module)),
	)

	if err != nil {
		return errors.Wrap(err, "failed to instantiate provision engine")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, pkg.Provision(engine))

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx := context.Background()
	ctx, _ = utils.WithSignal(ctx)

	// call the runtime upgrade before running engine
	provisioners.RuntimeUpgrade(ctx)

	// spawn the entine
	go func() {
		if err := engine.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("provision engine exited unexpectedely")
		}
	}()

	if err := app.MarkBooted(module); err != nil {
		log.Error().Err(err).Msg("failed to mark module as booted")
	}

	reporter, err := NewReported(store, identity, queues)
	if err != nil {
		return errors.Wrap(err, "failed to setup capacity reporter")
	}
	// also spawn the capacity reporter
	go func() {
		if err := reporter.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("capacity reported stopped unexpectedely")
		}
		log.Info().Msg("capacity reported stopped")
	}()

	// and start the zbus server in the back ground
	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("zbus provision engine api exited unexpectedely")
		}
		log.Info().Msg("zbus server stopped")
	}()

	if err := setupAPIs(v1, cl, engine); err != nil {
		return errors.Wrap(err, "failed to initialize API")
	}

	// register static files
	v1.PathPrefix("").Handler(http.StripPrefix("/api/v1", http.FileServer(http.FS(swaggerFs))))
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
		httpServer.Close()
	})

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "http api exited unexpectedely")
	}

	log.Info().Msg("provision engine stopped")
	return nil
}

func getNodeReserved(cl zbus.Client) (counter primitives.Counters, err error) {
	storage := stubs.NewStorageModuleStub(cl)
	fs, err := storage.GetCacheFS()
	if err != nil {
		return counter, err
	}

	var v *primitives.AtomicValue
	switch fs.DiskType {
	case zos.HDDDevice:
		v = &counter.HRU
	case zos.SSDDevice:
		v = &counter.SRU
	default:
		return counter, fmt.Errorf("unknown cache disk type '%s'", fs.DiskType)
	}

	v.Increment(fs.Usage.Size / gib)
	counter.MRU.Increment(2)
	return
}

func setupAPIs(v1 *mux.Router, cl zbus.Client, engine provision.Engine) error {

	_, err := api.NewWorkloadsAPI(v1, engine)
	if err != nil {
		return errors.Wrap(err, "failed to setup workload api")
	}

	_, err = api.NewNetworkAPI(v1, engine, cl)
	if err != nil {
		return errors.Wrap(err, "failed to setup network api")
	}

	return nil
}
