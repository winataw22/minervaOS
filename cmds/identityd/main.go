package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gedis"
	"github.com/threefoldtech/zos/pkg/geoip"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/identity"

	"github.com/threefoldtech/zos/pkg/zinit"

	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/upgrade"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	zinitSocket = "/var/run/zinit.sock"
)

const (
	module   = "identityd"
	seedName = "seed.txt"
)

// setup is a sanity check function, the whole purpose of this
// is to make sure at least required services are running in case
// of upgrade failure
// for example, in case of upgraded crash after it already stopped all
// the services for upgrade.
func setup(zinit *zinit.Client) error {
	for _, required := range []string{"redis", "flistd"} {
		if err := zinit.StartWait(5*time.Second, required); err != nil {
			return err
		}
	}

	return nil
}

// SafeUpgrade makes sure upgrade daemon is not interrupted
// While
func SafeUpgrade(upgrader *upgrade.Upgrader) error {
	ch := make(chan os.Signal)
	defer close(ch)
	defer signal.Stop(ch)

	// try to upgraded to latest
	// but mean while also make sure the daemon can not be killed by a signal
	signal.Notify(ch)
	return upgrader.Upgrade()
}

// This daemon startup has the follow flow:
// 1. Do upgrade to latest version (this might means it needs to restart itself)
// 2. Register the node to BCDB
// 3. start zbus server to serve identity interface
// 4. Start watcher for new version
// 5. On update, re-register the node with new version to BCDB

func main() {
	var (
		broker   string
		root     string
		interval int
		ver      bool
		debug    bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/identityd", "root working directory of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.IntVar(&interval, "interval", 600, "interval in seconds between update checks, default to 600")
	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.BoolVar(&debug, "d", false, "when set, no self update is done before upgradeing")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("failed to create root directory")
	}

	zinit, err := zinit.New(zinitSocket)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zinit")
	}

	cl, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Error().Err(err).Msg("fail to connect to broker")
		return
	}

	flister := stubs.NewFlisterStub(cl)

	upgrader := upgrade.Upgrader{
		FLister:      flister,
		Zinit:        zinit,
		NoSelfUpdate: debug,
	}

	bootMethod := upgrade.DetectBootMethod()

	// 2. Register the node to BCDB
	// at this point we are running latest version
	idMgr, err := identityMgr(root)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity manager")
	}

	idStore, err := bcdbClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity client")
	}

	var current string
	if bootMethod == upgrade.BootMethodFList {
		v, err := upgrader.Version()
		if err != nil {
			//NOTE: this is set to error intentionally (not fatal)
			//this will cause version to be 0.0.0 and will force an
			//immediate update of the flist to latest
			log.Error().Err(err).Msg("failed to read current version")
		}
		current = v.String()
	} else {
		current = "not booted from flist"
	}

	nodeID := idMgr.NodeID()
	farmID, err := idMgr.FarmID()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read farm ID")
	}

	loc, err := geoip.Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("fetch location")
	}

	register := func(v string) error {
		return registerNode(nodeID, farmID, v, idStore, loc)
	}

	if err := backoff.Retry(func() error {
		return register(current)
	}, backoff.NewExponentialBackOff()); err != nil {
		log.Error().Err(err).Msg("failed to register node")
	} else {
		log.Info().Str("version", current).Msg("node registered successfully")
	}

	// 3. start zbus server to serve identity interface
	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, idMgr)

	ctx, cancel := utils.WithSignal(context.Background())
	// register the cancel function with defer if the process stops because of a update
	defer cancel()

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}
	}()

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if bootMethod != upgrade.BootMethodFList {
		log.Info().Msg("node is not booted from an flist. upgrade is not supported")
		<-ctx.Done()
		return
	}

	// 4. Start watcher for new version
	log.Info().Msg("start upgrade daemon")
	ticker := time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()

	for {
		err := SafeUpgrade(&upgrader)
		if err == upgrade.ErrRestartNeeded {
			log.Info().Msg("restarting upgraded")
			return
		} else if err != nil {
			//TODO: crash or continue!
			log.Error().Err(err).Msg("upgrade failed")
		}

		version, err := upgrader.Version()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to read current version")
		}

		if version.String() != current {
			log.Info().Str("version", version.String()).Msg("new version")

			if err := backoff.Retry(func() error {
				return register(version.String())
			}, backoff.NewExponentialBackOff()); err != nil {
				log.Error().Err(err).Msg("failed to register node")
			} else {
				log.Info().Str("version", version.String()).Msg("node registered successfully")
			}

			current = version.String()
		}

		select {
		case <-ticker.C:
			log.Debug().Msg("checking for updates")
		case <-ctx.Done():
			return
		}
	}
}

func identityMgr(root string) (pkg.IdentityManager, error) {
	seedPath := filepath.Join(root, seedName)

	manager, err := identity.NewManager(seedPath)
	if err != nil {
		return nil, err
	}

	env := environment.Get()

	nodeID := manager.NodeID()
	log.Info().
		Str("identity", nodeID.Identity()).
		Msg("node identity loaded")

	log.Info().
		Bool("orphan", env.Orphan).
		Str("farmer_id", env.FarmerID).
		Msg("farmer identified")

	return manager, nil
}

// instantiate the proper client based on the running mode
func bcdbClient() (identity.IDStore, error) {
	env := environment.Get()

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return identity.NewHTTPIDStore(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbNamespace, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}
	return store, nil
}

func registerNode(nodeID, farmID pkg.Identifier, version string, store identity.IDStore, loc geoip.Location) error {
	log.Info().Str("version", version).Msg("start registration of the node")

	_, err := store.RegisterNode(nodeID, farmID, version, loc)
	if err != nil {
		return err
	}
	return nil
}
