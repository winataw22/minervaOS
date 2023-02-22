package provisiond

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rusart/muxprom"
	"github.com/threefoldtech/rmb-sdk-go"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/primitives"
	"github.com/threefoldtech/zos/pkg/provision/mbus"
	"github.com/threefoldtech/zos/pkg/provision/storage"
	fsStorage "github.com/threefoldtech/zos/pkg/provision/storage.fs"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	serverName       = "provision"
	provisionModule  = "provision"
	statisticsModule = "statistics"
	gib              = 1024 * 1024 * 1024

	boltStorageDB = "workloads.bolt"
	// old style rrd, make sure we clean it up
	metricsStorageDBOld = "metrics.bolt"
	// new style db after rrd implementation change
	metricsStorageDB = "metrics-diff.bolt"

	// deprecated, kept for migration
	fsStorageDB = "workloads"
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
	)

	ctx, _ := utils.WithSignal(context.Background())

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

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

	server, err := zbus.NewRedisServer(serverName, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "failed to connect to message broker")
	}
	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(cl)
	sk := ed25519.PrivateKey(identity.PrivateKey(ctx))

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
		return network.Ready(cli.Context)
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
	//v1 := router.PathPrefix("/api/v1").Subrouter()
	// keep track of resource units reserved and amount of workloads provisionned

	// to store reservation locally on the node
	store, err := storage.New(filepath.Join(rootDir, boltStorageDB))
	if err != nil {
		return errors.Wrap(err, "failed to create local reservation store")
	}
	defer store.Close()
	// we check if the old fs storage still exists
	fsStoragePath := filepath.Join(rootDir, fsStorageDB)
	if _, err := os.Stat(fsStoragePath); err == nil {
		// if it does we need to migrate this storage to new bolt storage
		fs, err := fsStorage.NewFSStore(fsStoragePath)
		if err != nil {
			return err
		}

		if err := storageMigration(store, fs); err != nil {
			return errors.Wrap(err, "storage migration failed")
		}

		if err := os.RemoveAll(fsStoragePath); err != nil {
			log.Error().Err(err).Msg("failed to clean up deprecated storage")
		}
	}

	if err := store.CleanDeleted(); err != nil {
		log.Error().Err(err).Msg("failed to purge deleted deployments history")
	}

	provisioners := primitives.NewPrimitivesProvisioner(cl)

	cap, err := capacity.NewResourceOracle(stubs.NewStorageModuleStub(cl)).Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	// update initial capacity with
	reserved, err := getNodeReserved(cl, cap)
	if err != nil {
		return errors.Wrap(err, "failed to get node reserved capacity")
	}
	var current gridtypes.Capacity
	var active []gridtypes.Deployment
	if !app.IsFirstBoot(serverName) {
		// if this is the first boot of this module.
		// it means the provision engine will still
		// rerun all deployments, which means we don't need
		// to set the current consumed capacity from store
		// since the counters will get populated anyway.
		// but if not, we need to set the current counters
		// from store.
		current, active, err = store.Capacity()
		if err != nil {
			log.Error().Err(err).Msg("failed to compute current consumed capacity")
		}
	}

	if err := netResourceMigration(active); err != nil {
		log.Error().Err(err).Msg("failed to migrate network resources")
	}

	log.Debug().Msgf("current used capacity: %+v", current)
	// statistics collects information about workload statistics
	// also does some checks on capacity
	statistics := primitives.NewStatistics(
		cap,
		store,
		reserved,
		provisioners,
	)

	mgr, err := environment.GetSubstrate()
	if err != nil {
		return err
	}
	users, err := provision.NewSubstrateTwins(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to create substrate users database")
	}

	admins, err := provision.NewSubstrateAdmins(mgr, uint32(env.FarmID))
	if err != nil {
		return errors.Wrap(err, "failed to create substrate admins database")
	}

	kp, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return errors.Wrap(err, "failed to get substrate keypair from secure key")
	}

	sub, err := mgr.Substrate()
	if err != nil {
		return errors.Wrap(err, "failed to connect to substrate")
	}
	defer sub.Close()
	twin, err := sub.GetTwinByPubKey(kp.PublicKey())
	if err != nil {
		return errors.Wrap(err, "failed to get node twin id")
	}

	node, err := sub.GetNodeByTwinID(twin)
	if err != nil {
		return errors.Wrap(err, "failed to get node from twin")
	}

	sub.Close()

	queues := filepath.Join(rootDir, "queues")
	if err := os.MkdirAll(queues, 0755); err != nil {
		return errors.Wrap(err, "failed to create storage for queues")
	}

	setter := NewCapacitySetter(kp, mgr, store)

	log.Info().Int("contracts", len(active)).Msg("setting used capacity by contracts")
	if err := setter.Set(active...); err != nil {
		log.Error().Err(err).Msg("failed to set capacity for active contracts")
	}

	log.Info().Msg("setting contracts used cpacity done")

	go func() {
		if err := setter.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("capacity reporter exited unexpectedly")
		}
	}()

	engine, err := provision.New(
		store,
		statistics,
		queues,
		provision.WithTwins(users),
		provision.WithAdmins(admins),
		provision.WithSubstrate(node, mgr),
		// set priority to some reservation types on boot
		// so we always need to make sure all volumes and networks
		// comes first.
		provision.WithStartupOrder(
			zos.ZMountType,
			zos.QuantumSafeFSType,
			zos.NetworkType,
			zos.PublicIPv4Type,
			zos.PublicIPType,
			zos.ZMachineType,
			zos.ZLogsType, //make sure zlogs comes after zmachine
		),
		// if this is a node reboot, the node needs to
		// recreate all reservations. so we set rerun = true
		provision.WithRerunAll(app.IsFirstBoot(serverName)),
		// Callback when a deployment changes capacity it must
		// be called. this one used by the setter to set used
		// capacity on chain.
		provision.WithCallback(setter.Callback),
	)

	if err != nil {
		return errors.Wrap(err, "failed to instantiate provision engine")
	}

	server.Register(
		zbus.ObjectID{Name: provisionModule, Version: "0.0.1"},
		pkg.Provision(engine),
	)

	server.Register(
		zbus.ObjectID{Name: statisticsModule, Version: "0.0.1"},
		pkg.Statistics(primitives.NewStatisticsStream(statistics)),
	)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	// call the runtime upgrade before running engine
	if err := provisioners.Initialize(ctx); err != nil {
		log.Error().Err(err).Msg("failed to run provisioners initializers")
	}

	// spawn the engine
	go func() {
		if err := engine.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("provision engine exited unexpectedely")
		}
	}()

	if err := app.MarkBooted(serverName); err != nil {
		log.Error().Err(err).Msg("failed to mark module as booted")
	}

	consumer, err := events.NewConsumer(msgBrokerCon, provisionModule)
	if err != nil {
		return errors.Wrap(err, "failed to create event consumer")
	}

	handler := NewContractEventHandler(node, mgr, engine, consumer)

	go func() {
		if err := handler.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("handling contracts events failed")
		}
	}()

	// clean up old rrd db that uses previous style reporting
	_ = os.Remove(filepath.Join(rootDir, metricsStorageDBOld))

	reporter, err := NewReporter(filepath.Join(rootDir, metricsStorageDB), cl, queues)
	if err != nil {
		return errors.Wrap(err, "failed to setup capacity reporter")
	}

	// also spawn the capacity reporter
	go func() {
		for {
			err := reporter.Run(ctx)
			if err == context.Canceled {
				return
			} else if err != nil {
				log.Error().Err(err).Msg("capacity reported stopped unexpectedely")
			}

			<-time.After(10 * time.Second)
		}
	}()

	// and start the zbus server in the background
	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("zbus provision engine api exited unexpectedely")
		}
		log.Info().Msg("zbus server stopped")
	}()

	mBus, err := rmb.NewRouter(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize message bus")
	}

	zosRouter := mBus.Subroute("zos")
	zosRouter.Use(rmb.LoggerMiddleware)
	// attach statistics module to rmb
	if err := primitives.NewStatisticsMessageBus(zosRouter, statistics); err != nil {
		return errors.Wrap(err, "failed to create statistics api")
	}

	setupStorageRmb(zosRouter, cl)

	_ = mbus.NewDeploymentMessageBus(zosRouter, engine)

	log.Info().Msg("running rmb handler")

	for _, handler := range mBus.Handlers() {
		log.Debug().Msgf("registered handler: %s", handler)
	}

	if err := mBus.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "message bus error")
	}

	log.Info().Msg("provision engine stopped")
	return nil
}

func getNodeReserved(cl zbus.Client, available gridtypes.Capacity) (counter gridtypes.Capacity, err error) {
	// fill in reserved storage
	storage := stubs.NewStorageModuleStub(cl)
	fs, err := storage.Cache(context.TODO())
	if err != nil {
		return counter, err
	}

	counter.SRU += fs.Usage.Size
	counter.MRU += gridtypes.Max(
		available.MRU*10/100,
		2*gridtypes.Gigabyte,
	)

	return
}

func setupStorageRmb(router rmb.Router, cl zbus.Client) {
	storage := router.Subroute("storage")
	storage.WithHandler("pools", func(ctx context.Context, payload []byte) (interface{}, error) {
		stub := stubs.NewStorageModuleStub(cl)
		return stub.Metrics(ctx)
	})
}
