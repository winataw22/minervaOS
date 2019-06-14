package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cenkalti/backoff"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/zinit"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

var root = flag.String("root", "/var/modules/network", "root path of the module")
var broker = flag.String("broker", redisSocket, "connection string to broker")

func main() {
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := backoff.Retry(bootstrap, backoff.NewExponentialBackOff())
	if err != nil {
		return
	}

	if err := os.MkdirAll(*root, 0750); err != nil {
		log.Fatal().Msgf("fail to create module root: %s", err)
	}

	netAlloc := network.NewTestNetResourceAllocator()
	networker := network.NewNetworker(netAlloc)

	server, err := zbus.NewRedisServer(module, *broker, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", *broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
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
