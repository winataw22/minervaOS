package provisiond

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/provision/explorer"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
	"github.com/threefoldtech/zos/pkg/provision/primitives/cache"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	module = "provision"
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
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		storageDir   string = cli.String("root")
	)

	flag.StringVar(&storageDir, "root", "/var/cache/modules/provisiond", "root path of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")

	// keep checking if limited-cache flag is set
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("failed cache reservation! Retrying every 30 seconds...")
		for app.CheckFlag(app.LimitedCache) {
			time.Sleep(time.Second * 30)
		}
	}

	if err := os.MkdirAll(storageDir, 0770); err != nil {
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
	zbusCl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(zbusCl)
	nodeID := identity.NodeID()

	// block until networkd is ready to serve request from zbus
	// this is used to prevent uptime and online status to the explorer if the node is not in a fully ready
	// https://github.com/threefoldtech/zos/issues/632
	network := stubs.NewNetworkerStub(zbusCl)
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	backoff.RetryNotify(func() error {
		return network.Ready()
	}, bo, func(err error, d time.Duration) {
		log.Error().Err(err).Msg("networkd is not ready yet")
	})

	// to get reservation from tnodb
	e, err := app.ExplorerClient()
	if err != nil {
		return errors.Wrap(err, "failed to instantiate BCDB client")
	}

	// keep track of resource units reserved and amount of workloads provisionned
	statser := &primitives.Counters{}

	// to store reservation locally on the node
	localStore, err := cache.NewFSStore(filepath.Join(storageDir, "reservations"))
	if err != nil {
		return errors.Wrap(err, "failed to create local reservation store")
	}

	// update stats from the local reservation cache
	localStore.Sync(statser)

	provisioner := primitives.NewProvisioner(localStore, zbusCl)

	puller := explorer.NewPoller(e, primitives.WorkloadToProvisionType, primitives.ProvisionOrder)
	engine, err := provision.New(provision.EngineOps{
		NodeID: nodeID.Identity(),
		Cache:  localStore,
		Source: provision.CombinedSource(
			provision.PollSource(puller, nodeID),
			provision.NewDecommissionSource(localStore),
		),
		Provisioners:   provisioner.Provisioners,
		Decomissioners: provisioner.Decommissioners,
		Feedback:       explorer.NewFeedback(e, primitives.ResultToSchemaType),
		Signer:         identity,
		Statser:        statser,
		ZbusCl:         zbusCl,
		Janitor:        provision.NewJanitor(zbusCl, puller),
	})

	if err != nil {
		return errors.Wrap(err, "failed to instantiate provision engine")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, pkg.Provision(engine))

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx := context.Background()
	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// call the runtime upgrade before running engine
	provisioner.RuntimeUpgrade(ctx)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
		log.Info().Msg("zbus server stopped")
	}()

	if err := engine.Run(ctx); err != nil {
		return errors.Wrap(err, "unexpected error")
	}

	log.Info().Msg("provision engine stopped")
	return nil
}

type store interface {
	provision.ReservationPoller
	provision.Feedbacker
}
