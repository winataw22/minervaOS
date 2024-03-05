package networkd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/dhcp"
	"github.com/threefoldtech/zos/pkg/network/public"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/urfave/cli/v2"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "network"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "networkd",
	Usage: "handles network resources and user networks",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/networkd",
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
		root   string = cli.String("root")
		broker string = cli.String("broker")
	)

	if err := os.MkdirAll(root, 0755); err != nil {
		return errors.Wrap(err, "fail to create module root")
	}

	waitYggdrasilBin()

	if err := migrateOlderDHCPService(); err != nil {
		return errors.Wrap(err, "failed to migrate older dhcp service")
	}

	if err := bootstrap.DefaultBridgeValid(); err != nil {
		return errors.Wrap(err, "invalid setup")
	}

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID(cli.Context)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := ensureHostFw(ctx); err != nil {
		return errors.Wrap(err, "failed to host firewall rules")
	}

	public.SetPersistence(root)

	pub, err := public.LoadPublicConfig()
	log.Debug().Err(err).Msgf("public interface configured: %+v", pub)
	if err != nil && err != public.ErrNoPublicConfig {
		return errors.Wrap(err, "failed to get node public_config")
	}

	// EnsurePublicSetup knows how to handle a nil pub (in case of ErrNoPublicConfig)
	master, err := public.EnsurePublicSetup(nodeID, environment.MustGet().PubVlan, pub)
	if err != nil {
		return errors.Wrap(err, "failed to setup public bridge")
	}

	dmz := ndmz.New(nodeID.Identity(), master)

	if err := dmz.Create(ctx); err != nil {
		return errors.Wrap(err, "failed to create ndmz")
	}
	log.Debug().Msg("starting yggdrasil")
	yggNamespace := dmz.Namespace()
	if public.HasPublicSetup() {
		yggNamespace = public.PublicNamespace
	}

	yggNs, err := yggdrasil.NewYggdrasilNamespace(yggNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to create yggdrasil namespace")
	}

	ygg, err := yggdrasil.EnsureYggdrasil(ctx, identity.PrivateKey(cli.Context), yggNs)
	if err != nil {
		return errors.Wrap(err, "failed to start yggdrasil")
	}

	if public.HasPublicSetup() {
		// if yggdrasil is living inside public namespace
		// we still need to setup ndmz to also have yggdrasil but we set the yggdrasil interface
		// a different Ip that lives inside the yggdrasil range.
		dmzYgg, err := yggdrasil.NewYggdrasilNamespace(dmz.Namespace())
		if err != nil {
			return errors.Wrap(err, "failed to setup ygg for dmz namespace")
		}

		ip, err := ygg.SubnetFor([]byte(fmt.Sprintf("ygg:%s", dmz.Namespace())))
		if err != nil {
			return errors.Wrap(err, "failed to calculate ip for ygg inside dmz")
		}

		gw, err := ygg.Gateway()
		if err != nil {
			return err
		}

		if err := dmzYgg.SetYggIP(ip, gw.IP); err != nil {
			return errors.Wrap(err, "failed to set yggdrasil ip for dmz")
		}
	}

	networker, err := network.NewNetworker(identity, dmz, ygg)
	if err != nil {
		return errors.Wrap(err, "error creating network manager")
	}

	mBus, err := rmb.NewRouter(broker)
	if err != nil {
		return errors.Wrap(err, "failed to initialize message bus")
	}

	zosRouter := mBus.Subroute("zos")
	zosRouter.Use(rmb.LoggerMiddleware)

	if _, err := network.NewNetworkMessageBus(zosRouter, networker, stubs.NewAPIGatewayStub(client)); err != nil {
		return errors.Wrap(err, "failed to initialize rmb api")
	}

	// we need to start both rmb server and zbus server.
	go func(ctx context.Context) {
		if err := startRmbServer(ctx, mBus); err != nil {
			log.Fatal().Err(err).Msg("rmb exited unexpectedly")
		}
	}(ctx)

	log.Info().Msg("start zbus server")
	if err := startZBusServer(ctx, broker, networker); err != nil {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}

func startRmbServer(ctx context.Context, bus *rmb.DefaultRouter) error {
	log.Info().
		Msg("starting networkd rmb module")

	for _, handler := range bus.Handlers() {
		log.Debug().Msgf("registered handler: %s", handler)
	}

	if err := bus.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

func startZBusServer(ctx context.Context, broker string, networker pkg.Networker) error {

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Error().Err(err).Msgf("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

func waitYggdrasilBin() {
	log.Info().Msg("wait for yggdrasil binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 //forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("yggdrasil")
		return err
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("yggdrasil binary not found, retying in %s", d.String())
	})
}

func migrateOlderDHCPService() error {
	// only migrate dhcp service if it's using the older client
	dhcpService := dhcp.NewService(types.DefaultBridge, "", zinit.Default())
	if dhcpService.IsUsingOlderClient() {
		// only migrate if it's using older client
		if err := dhcpService.Destroy(); err != nil {
			return errors.Wrapf(err, "failed to destory older dhcp service")
		}
		if err := dhcpService.Create(); err != nil {
			return errors.Wrapf(err, "failed to create newer dhcp service")
		}
		return dhcpService.Start()
	}

	return nil
}
