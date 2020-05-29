package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jbenet/go-base58"
	"github.com/shirou/gopsutil/host"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/flist"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/upgrade"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/identity"

	"github.com/threefoldtech/zos/pkg/zinit"

	"flag"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
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
	for _, required := range []string{"redis"} {
		if err := zinit.StartWait(5*time.Second, required); err != nil {
			return err
		}
	}

	return nil
}

// Safe makes sure function call not interrupted
// with a signal while exection
func Safe(fn func() error) error {
	ch := make(chan os.Signal)
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
		client, err := zbus.NewRedisClient(broker)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to zbus")
		}
		stub := stubs.NewIdentityManagerStub(client)
		nodeID := stub.NodeID()
		fmt.Println(nodeID)
		os.Exit(0)
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("failed to create root directory")
	}

	boot := upgrade.Boot{}

	bootMethod := boot.DetectBootMethod()

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
		v, err := boot.Version()
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

	if err := backoff.RetryNotify(func() error {
		return register(current)
	}, backoff.NewExponentialBackOff(), retryNotify); err != nil {
		log.Error().Err(err).Msg("failed to register node")
	} else {
		log.Info().Str("version", current).Msg("node registered successfully")
	}

	monitor := newVersionMonitor(2 * time.Second)
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

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}
	}()

	zinit, err := zinit.New(zinitSocket)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zinit")
	}

	// with volume allocation is set to nil this flister can be
	// only used for RO mounts. Otherwise it will panic
	flister := flist.New(root, nil)

	upgrader := upgrade.Upgrader{
		FLister:      flister,
		Zinit:        zinit,
		NoSelfUpdate: debug,
	}

	installBinaries(&boot, &upgrader)

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

	upgradeLoop(ctx, &boot, &upgrader, debug, monitor, register)
}

func getBinsRepo() string {
	env, _ := environment.Get()

	switch env.RunningMode {
	case environment.RunningDev:
		return "tf-zos-bins.dev"
	case environment.RunningTest:
		return "tf-zos-bins.test"
	default:
		return "tf-zos-bins"
	}
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

func installBinaries(boot *upgrade.Boot, upgrader *upgrade.Upgrader) {

	bins, _ := boot.CurrentBins()

	repoWatcher := upgrade.FListRepoWatcher{
		Repo:    getBinsRepo(),
		Current: bins,
	}

	current, toAdd, toDel, err := repoWatcher.Diff()
	if err != nil {
		log.Error().Err(err).Msg("failed to list latest binaries to install")
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

	boot.SetBins(current)
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

	flistWatcher := upgrade.FListSemverWatcher{
		FList:    boot.Name(),
		Current:  boot.MustVersion(), //if we are here version must be valid
		Duration: 600 * time.Second,
	}

	// make sure we push the current version to monitor
	monitor.C <- boot.MustVersion()

	flistEvents, err := flistWatcher.Watch(ctx)
	if err != nil {
		log.Fatal().Err(err).Str("flist", flistWatcher.FList).Msg("failed to watch flist")
	}

	bins, err := boot.CurrentBins()
	if err != nil {
		log.Warn().Err(err).Msg("could not load current binaries list")
	}

	repoWatcher := upgrade.FListRepoWatcher{
		Repo:    getBinsRepo(),
		Current: bins,
	}

	repoEvents, err := repoWatcher.Watch(ctx)
	if err != nil {
		log.Fatal().Err(err).Str("repo", repoWatcher.Repo).Msg("failed to watch repo")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-flistEvents:
			if e == nil {
				continue
			}
			event := e.(*upgrade.FListEvent)
			// new flist available
			version, err := event.Version()
			if err != nil {
				log.Error().Err(err).Msg("failed to parse new version")
				continue
			}

			from, err := boot.Current()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load current boot information")
				return
			}

			err = Safe(func() error {
				return upgrader.Upgrade(from, *event)
			})

			if err == upgrade.ErrRestartNeeded {
				log.Info().Msg("restarting upgraded")
				return
			} else if err != nil {
				//TODO: crash or continue!
				log.Error().Err(err).Msg("upgrade failed")
				continue
			}

			if err := boot.Set(*event); err != nil {
				log.Error().Err(err).Msg("failed to update boot information")
			}

			monitor.C <- version
			if err := backoff.RetryNotify(func() error {
				return register(version.String())
			}, backoff.NewExponentialBackOff(), retryNotify); err != nil {
				log.Error().Err(err).Msg("failed to register node")
			} else {
				log.Info().Str("version", version.String()).Msg("node registered successfully")
			}
		case e := <-repoEvents:
			if e == nil {
				continue
			}
			event := e.(*upgrade.RepoEvent)
			for _, bin := range event.ToDel {
				if err := upgrader.UninstallBinary(bin); err != nil {
					log.Error().Err(err).Str("flist", bin.Fqdn()).Msg("failed to uninstall flist")
				}
			}

			for _, bin := range event.ToAdd {
				if err := upgrader.InstallBinary(bin); err != nil {
					log.Error().Err(err).Str("flist", bin.Fqdn()).Msg("failed to install flist")
				}
			}

			if err := boot.SetBins(repoWatcher.Current); err != nil {
				log.Error().Err(err).Msg("failed to update local db of installed binaries")
			}

			log.Debug().Msg("finish processing binary updates")
		}
	}
}

func retryNotify(err error, d time.Duration) {
	log.Warn().Err(err).Str("sleep", d.String()).Msg("registration failed")
}

func identityMgr(root string) (pkg.IdentityManager, error) {
	seedPath := filepath.Join(root, seedName)

	manager, err := identity.NewManager(seedPath)
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

// instantiate the proper client based on the running mode
func bcdbClient() (client.Directory, error) {
	client, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	return client.Directory, nil
}

func registerNode(nodeID pkg.Identifier, farmID pkg.FarmID, version string, store client.Directory, loc geoip.Location) error {
	log.Info().Str("version", version).Msg("start registration of the node")

	v1ID, _ := network.NodeIDv1()

	publicKeyHex := hex.EncodeToString(base58.Decode(nodeID.Identity()))

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unknown"
	}

	uptime, err := hostUptime()
	if err != nil {
		return errors.Wrap(err, "could not get node uptime")
	}

	err = store.NodeRegister(directory.Node{
		NodeId:    nodeID.Identity(),
		HostName:  hostName,
		NodeIdV1:  v1ID,
		FarmId:    int64(farmID),
		OsVersion: version,
		Location: directory.Location{
			Continent: loc.Continent,
			Country:   loc.Country,
			City:      loc.City,
			Longitude: loc.Longitute,
			Latitude:  loc.Latitude,
		},
		PublicKeyHex: publicKeyHex,
		Uptime:       int64(uptime),
	})

	if err != nil {
		log.Error().Err(err).Send()
		return err
	}

	return nil
}

// hostUptime returns the uptime of the node
func hostUptime() (uint64, error) {
	info, err := host.Info()
	if err != nil {
		return 0, err
	}
	return info.Uptime, nil
}
