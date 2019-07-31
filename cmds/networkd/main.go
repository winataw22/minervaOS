package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/version"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"

	"github.com/cenkalti/backoff"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/threefoldtech/zosv2/modules/zinit"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

func main() {
	var (
		tnodbURL string
		root     string
		broker   string
		ver      bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/network", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus broker")
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Error().Err(err).Msgf("fail to create module root")
	}

	db := tnodb.NewHTTPHTTPTNoDB(tnodbURL)

	if err := ifaceutil.SetLoUp(); err != nil {
		return
	}

	if err := bootstrap(); err != nil {
		log.Error().Err(err).Msg("failed to bootstrap network")
		os.Exit(1)
	}

	log.Info().Msg("network bootstraped successfully")

	if err := ready(); err != nil {
		log.Fatal().Err(err).Msg("failed to mark networkd as ready")
	}

	identity := stubs.NewIdentityManagerStub(client)
	networker := network.NewNetworker(identity, db, root)
	nodeID := identity.NodeID()

	if err := publishIfaces(nodeID, db); err != nil {
		log.Error().Err(err).Msg("failed to publish network interfaces to tnodb")
		os.Exit(1)
	}

	ifaceVersion := -1
	exitIface, err := db.ReadPubIface(nodeID)
	if err == nil {
		if err := configuePubIface(exitIface); err != nil {
			log.Error().Err(err).Msg("failed to configure public interface")
			os.Exit(1)
		}
		ifaceVersion = exitIface.Version
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chIface := watchPubIface(ctx, nodeID, db, ifaceVersion)
	go func(ctx context.Context, ch <-chan *network.PubIface) {
		for {
			select {
			case iface := <-ch:
				_ = configuePubIface(iface)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, chIface)

	if err := startServer(ctx, broker, networker); err != nil {
		log.Error().Err(err).Msg("fail to start networkd")
	}
}

func bootstrap() error {
	f := func() error {

		z := zinit.New("")
		if err := z.Connect(); err != nil {
			log.Error().Err(err).Msg("fail to connect to zinit")
			return err
		}

		log.Info().Msg("Start network bootstrap")
		if err := network.Bootstrap(); err != nil {
			log.Error().Err(err).Msg("fail to boostrap network")
			return err
		}

		log.Info().Msg("writing udhcp init service")

		err := zinit.AddService("dhcp_zos", zinit.InitService{
			Exec:    fmt.Sprintf("/sbin/udhcpc -v -f -i %s -s /usr/share/udhcp/simple.script", network.DefaultBridge),
			Oneshot: false,
			After:   []string{},
		})
		if err != nil {
			log.Error().Err(err).Msg("fail to create dhcp_zos zinit service")
			return err
		}

		if err := z.Monitor("dhcp_zos"); err != nil {
			log.Error().Err(err).Msg("fail to start monitoring dhcp_zos zinit service")
			return err
		}
		return nil
	}

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to bootstrap network")
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}

func publishIfaces(id modules.Identifier, db network.TNoDB) error {
	f := func() error {
		log.Info().Msg("try to publish interfaces to TNoDB")
		return db.PublishInterfaces(id)
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

	return server.Run(context.Background())
}

func ready() error {
	f, err := os.Create("/var/run/networkd.ready")
	defer f.Close()
	return err
}
