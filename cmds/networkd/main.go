package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/identity"

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
	)

	flag.StringVar(&root, "root", "/var/modules/network", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")

	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := ifaceutil.SetLoUp(); err != nil {
		return
	}

	err := backoff.Retry(bootstrap, backoff.NewExponentialBackOff())
	if err != nil {
		return
	}

	time.Sleep(5 * time.Second)

	db := tnodb.NewHTTPHTTPTNoDB(tnodbURL)
	f := func() error {
		log.Info().Msg("try to publish interfaces to TNoDB")
		return db.PublishInterfaces()
	}

	err = backoff.RetryNotify(f, backoff.NewExponentialBackOff(), func(error, time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to publish the node interaces")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to publish interfaces to TNoDB")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := watchPubIface(ctx, db)
	go func(ctx context.Context, ch <-chan *network.PubIface) {
		for {
			select {
			case iface := <-ch:
				_ = configuePubIface(iface)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, ch)

	startServer(ctx, root, broker, db)
}

func bootstrap() error {
	z := zinit.New("")
	if err := z.Connect(); err != nil {
		return err
	}

	log.Info().Msg("Start network bootstrap")
	if err := network.Bootstrap(); err != nil {
		return err
	}

	log.Info().Msg("writing udhcp init service")

	err := zinit.AddService("dhcp_zos", zinit.InitService{
		Exec:    fmt.Sprintf("/sbin/udhcpc -v -f -i %s -s /usr/share/udhcp/simple.script", network.DefaultBridgeName()),
		Oneshot: false,
		After:   []string{},
	})
	if err != nil {
		return err
	}

	return z.Monitor("dhcp_zos")
}

func startServer(ctx context.Context, root, broker string, db network.TNoDB) error {
	if err := os.MkdirAll(root, 0750); err != nil {
		log.Error().Err(err).Msgf("fail to create module root")
	}

	nodeID, err := identity.LocalNodeID()
	if err != nil {
		return err
	}

	networker := network.NewNetworker(ctx, nodeID, db, root)

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
