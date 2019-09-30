package main

import (
	"context"
	"flag"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/environment"
	"github.com/threefoldtech/zosv2/modules/gedis"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/ndmz"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/utils"
	"github.com/threefoldtech/zosv2/modules/version"
	"github.com/vishvananda/netlink"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

func main() {
	var (
		root   string
		broker string
		ver    bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/networkd", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if err := network.DefaultBridgeValid(); err != nil {
		log.Fatal().Err(err).Msg("invalid setup")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus broker")
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Msgf("fail to create module root")
	}

	db, err := bcdbClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to BCDB")
	}

	identity := stubs.NewIdentityManagerStub(client)
	networker := network.NewNetworker(identity, db, root)
	nodeID := identity.NodeID()

	if err := publishIfaces(nodeID, db); err != nil {
		log.Error().Err(err).Msg("failed to publish network interfaces to tnodb")
		os.Exit(1)
	}

	ifaceVersion := -1
	exitIface, err := db.GetPubIface(nodeID)
	if err == nil {
		if err := configurePubIface(exitIface); err != nil {
			log.Error().Err(err).Msg("failed to configure public interface")
			os.Exit(1)
		}
		ifaceVersion = exitIface.Version
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chIface := watchPubIface(ctx, nodeID, db, ifaceVersion)
	go func(ctx context.Context, ch <-chan *types.PubIface) {
		for {
			select {
			case iface := <-ch:
				_ = configurePubIface(iface)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, chIface)

	if err := ndmz.Create(); err != nil {
		log.Fatal().Err(err).Msgf("failed to create DMZ")
	}

	if err := startServer(ctx, broker, networker); err != nil {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}

func getLocalInterfaces() ([]types.IfaceInfo, error) {
	var output []types.IfaceInfo

	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return nil, err
	}

	for _, link := range ifaceutil.LinkFilter(links, []string{"device", "bridge"}) {
		//TODO: why bringing the interface up? shouldn't we just check if it's up or not?
		if err := netlink.LinkSetUp(link); err != nil {
			log.Info().Str("interface", link.Attrs().Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(link.Attrs().Name) && !ifaceutil.IsPluggedTimeout(link.Attrs().Name, time.Second*5) {
			log.Info().Str("interface", link.Attrs().Name).Msg("interface is not plugged in, skipping")
			continue
		}

		_, gw, err := ifaceutil.HasDefaultGW(link)
		if err != nil {
			return nil, err
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return nil, err
		}

		info := types.IfaceInfo{
			Name:  link.Attrs().Name,
			Addrs: make([]*net.IPNet, len(addrs)),
		}
		for i, addr := range addrs {
			info.Addrs[i] = addr.IPNet
		}

		if gw != nil {
			info.Gateway = append(info.Gateway, gw)
		}

		output = append(output, info)
	}

	return output, err
}

func publishIfaces(id modules.Identifier, db network.TNoDB) error {
	ifaces, err := getLocalInterfaces()
	if err != nil {
		return err
	}

	f := func() error {
		log.Info().Msg("try to publish interfaces to TNoDB")
		return db.PublishInterfaces(id, ifaces)
	}
	errHandler := func(err error, _ time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to publish the node interaces")
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}

func startServer(ctx context.Context, broker string, networker modules.Networker) error {

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Error().Err(err).Msgf("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

// instantiate the proper client based on the running mode
func bcdbClient() (network.TNoDB, error) {
	env := environment.Get()

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return tnodb.NewHTTPTNoDB(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbNamespace, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}
	return store, nil
}
