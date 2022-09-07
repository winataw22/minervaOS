package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/upgrade"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/identity"

	"flag"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
)

const (
	module = "identityd"
)

// Safe makes sure function call not interrupted
// with a signal while exection
func Safe(fn func() error) error {
	ch := make(chan os.Signal, 4)
	defer close(ch)
	defer signal.Stop(ch)

	// try to upgraded to latest
	// but mean while also make sure the daemon can not be killed by a signal
	signal.Notify(ch)
	return fn()
}

// This daemon startup has the follow flow:
// 1. Do upgrade to latest version (this might means it needs to restart itself)
// 2. Register the node to BCDB
// 3. start zbus server to serve identity interface
// 4. Start watcher for new version
// 5. On update, re-register the node with new version to BCDB

func main() {
	app.Initialize()

	var (
		broker   string
		root     string
		interval int
		ver      bool
		debug    bool
		id       bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/identityd", "root working directory of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.IntVar(&interval, "interval", 600, "interval in seconds between update checks, default to 600")
	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.BoolVar(&debug, "d", false, "when set, no self update is done before upgradeing")
	flag.BoolVar(&id, "id", false, "prints the node ID and exits")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if id {
		ctx := context.Background()
		client, err := zbus.NewRedisClient(broker)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to zbus")
		}
		stub := stubs.NewIdentityManagerStub(client)
		nodeID := stub.NodeID(ctx)
		fmt.Println(nodeID)
		os.Exit(0)
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("failed to create root directory")
	}

	var boot upgrade.Boot

	bootMethod := boot.DetectBootMethod()

	// 2. Register the node to BCDB
	// at this point we are running latest version
	idMgr, err := getIdentityMgr(root, debug)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity manager")
	}

	monitor := newVersionMonitor(10 * time.Second)
	// 3. start zbus server to serve identity interface
	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, idMgr)
	server.Register(zbus.ObjectID{Name: "monitor", Version: "0.0.1"}, monitor)

	ctx, cancel := utils.WithSignal(context.Background())
	// register the cancel function with defer if the process stops because of a update
	defer cancel()

	upgrader, err := upgrade.NewUpgrader(root, upgrade.NoSelfUpgrade(debug))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize upgrader")
	}

	err = installBinaries(&boot, upgrader)
	if err == upgrade.ErrRestartNeeded {
		return
	} else if err != nil {
		log.Error().Err(err).Msg("failed to install binaries")
	}

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}
	}()

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("received a termination signal")
	})

	if bootMethod != upgrade.BootMethodFList {
		log.Info().Msg("node is not booted from an flist. upgrade is not supported")
		<-ctx.Done()
		return
	}

	//NOTE: code after this commit will only
	//run if the system is booted from an flist

	// 4. Start watcher for new version
	log.Info().Msg("start upgrade daemon")

	// TODO: do we need to update farmer on node upgrade?
	upgradeLoop(ctx, &boot, upgrader, debug, monitor, func(string) error { return nil })
}

// allow reinstall if receive signal USR1
// only allowed in debug mode
func debugReinstall(boot *upgrade.Boot, up *upgrade.Upgrader) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1)

	go func() {
		for range c {
			current, err := boot.Current()
			if err != nil {
				log.Error().Err(err).Msg("couldn't get current flist info")
				continue
			}

			if err := Safe(func() error {
				return up.Upgrade(current, current)
			}); err != nil {
				log.Error().Err(err).Msg("reinstall failed")
			} else {
				log.Info().Msg("reinstall completed successfully")
			}
		}
	}()
}

func installBinaries(boot *upgrade.Boot, upgrader *upgrade.Upgrader) error {
	bins, _ := boot.CurrentBins()
	env, _ := environment.Get()

	repoWatcher := upgrade.FListRepo{
		Repo:    env.BinRepo,
		Current: bins,
	}

	current, toAdd, toDel, err := repoWatcher.Diff()
	if err != nil {
		return errors.Wrap(err, "failed to list latest binaries to install")
	}

	if len(toAdd) == 0 && len(toDel) == 0 {
		return nil
	}

	for _, pkg := range toDel {
		if err := upgrader.UninstallBinary(pkg); err != nil {
			log.Error().Err(err).Str("flist", pkg.Fqdn()).Msg("failed to uninstall flist")
		}
	}

	for _, pkg := range toAdd {
		if err := upgrader.InstallBinary(pkg); err != nil {
			log.Error().Err(err).Str("package", pkg.Fqdn()).Msg("failed to install package")
		}
	}

	if err := boot.SetBins(current); err != nil {
		return errors.Wrap(err, "failed to commit pkg status")
	}

	return upgrade.ErrRestartNeeded
}

func upgradeLoop(
	ctx context.Context,
	boot *upgrade.Boot,
	upgrader *upgrade.Upgrader,
	debug bool,
	monitor *monitorStream,
	register func(string) error) {

	if debug {
		debugReinstall(boot, upgrader)
	}

	monitor.C <- boot.MustVersion()
	var hub upgrade.HubClient

	flist := boot.Name()
	//current := boot.MustVersion()
	for {
		// delay in case of error
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}

		current, err := boot.Current()
		if err != nil {
			log.Fatal().Err(err).Msg("cannot get current boot flist information")
		}

		latest, err := hub.Info(flist)
		if err != nil {
			log.Error().Err(err).Msg("failed to get flist info")
			continue
		}
		log.Info().
			Str("current", current.TryVersion().String()).
			Str("latest", current.TryVersion().String()).
			Msg("checking if update is required")

		if !latest.TryVersion().GT(current.TryVersion()) {
			// We wanted to use the node id to actually calculate the delay to wait but it's not
			// possible to get the numeric node id from the identityd
			next := time.Duration(60+rand.Intn(60)) * time.Minute
			log.Info().Dur("wait", next).Msg("checking for update after milliseconds")
			select {
			case <-time.After(next):
			case <-ctx.Done():
				return
			}

			continue
		}

		err = installBinaries(boot, upgrader)
		if err == upgrade.ErrRestartNeeded {
			log.Info().Msg("restarting upgraded")
			return
		} else if err != nil {
			log.Error().Err(err).Msg("failed to update runtime binaries")
		}

		// next check for update
		exp := backoff.NewExponentialBackOff()
		exp.MaxInterval = 3 * time.Minute
		exp.MaxElapsedTime = 60 * time.Minute
		err = backoff.Retry(func() error {
			log.Debug().Str("version", latest.TryVersion().String()).Msg("trying to update")

			err := Safe(func() error {
				return upgrader.Upgrade(current, latest)
			})

			if err == upgrade.ErrRestartNeeded {
				return backoff.Permanent(err)
			} else if err != nil {
				log.Error().Err(err).Msg("update failure. retrying")
			}

			return nil
		}, exp)

		if err == upgrade.ErrRestartNeeded {
			log.Info().Msg("restarting upgraded")
			return
		} else if err != nil {
			//TODO: crash or continue!
			log.Error().Err(err).Msg("upgrade failed")
			continue
		} else {
			log.Info().Str("version", latest.TryVersion().String()).Msg("update completed")
		}
		if err := boot.Set(latest); err != nil {
			log.Fatal().Err(err).Msg("failed to update boot information")
		}

		monitor.C <- latest.TryVersion()
	}
}

func getIdentityMgr(root string, debug bool) (pkg.IdentityManager, error) {
	manager, err := identity.NewManager(root, debug)
	if err != nil {
		return nil, err
	}

	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse node environment")
	}

	nodeID := manager.NodeID()
	log.Info().
		Str("identity", nodeID.Identity()).
		Msg("node identity loaded")

	log.Info().
		Bool("orphan", env.Orphan).
		Uint64("farmer_id", uint64(env.FarmerID)).
		Msg("farmer identified")

	return manager, nil
}
