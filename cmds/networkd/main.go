package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gedis"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/tnodb"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
	"github.com/vishvananda/netlink"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

func main() {
	app.Initialize()

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
		if err := configurePubIface(exitIface, nodeID); err != nil {
			log.Error().Err(err).Msg("failed to configure public interface")
			os.Exit(1)
		}
		ifaceVersion = exitIface.Version
	}

	if err := ndmz.Create(nodeID); err != nil {
		log.Fatal().Err(err).Msgf("failed to create DMZ")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chIface := watchPubIface(ctx, nodeID, db, ifaceVersion)
	go func(ctx context.Context, ch <-chan *types.PubIface) {
		for {
			select {
			case iface := <-ch:

				// When changing public IP configuration, we need to also update how the NDMZ interfaces are plumbed together
				// to achieve this any time a new public config is received, we first delete the NDMZ namespace
				// create/update the public namespace and then re-create the NDMZ

				log.Info().Str("config", fmt.Sprintf("%+v", iface)).Msg("public IP configuration received")

				if err := ndmz.Delete(); err != nil {
					log.Error().Err(err).Msg("error deleting ndmz during public ip configuration")
					continue
				}

				if err = configurePubIface(iface, nodeID); err != nil {
					log.Error().Err(err).Msg("error configuring public IP")
					continue
				}

				if err := ndmz.Create(nodeID); err != nil {
					log.Error().Err(err).Msg("error configuring ndmz during public ip configuration")
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, chIface)

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
			Addrs: make([]types.IPNet, len(addrs)),
		}
		for i, addr := range addrs {
			info.Addrs[i] = types.NewIPNet(addr.IPNet)
		}

		if gw != nil {
			info.Gateway = append(info.Gateway, gw)
		}

		output = append(output, info)
	}

	return output, err
}

func publishIfaces(id pkg.Identifier, db network.TNoDB) error {
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

func startServer(ctx context.Context, broker string, networker pkg.Networker) error {

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
	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse node environment")
	}

	// use the bcdb mock for dev and test
	if env.RunningMode == environment.RunningDev {
		return tnodb.NewHTTPTNoDB(env.BcdbURL), nil
	}

	// use gedis for production bcdb
	store, err := gedis.New(env.BcdbURL, env.BcdbPassword)
	if err != nil {
		return nil, errors.Wrap(err, "fail to connect to BCDB")
	}
	return store, nil
}
