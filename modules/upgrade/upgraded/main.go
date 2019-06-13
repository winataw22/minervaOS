package main

import (
	"os"
	"time"

	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/upgrade"
)

const redisSocket = "unix:///var/run/redis.sock"

var root = flag.String("root", "/var/modules/upgrade", "root path of the module")
var broker = flag.String("broker", redisSocket, "connection string to broker")

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	flag.Parse()

	zbusClient, err := zbus.NewRedisClient(*broker)
	if err != nil {
		log.Error().Err(err).Msg("fail to connect to broker")
		return
	}

	flister := stubs.NewFlisterStub(zbusClient)

	u, err := upgrade.New(*root, flister)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to instantiate upgrade module")
	}

	// watcher := upgrade.NewPeriodicWatcher(10 * time.Second)
	publisher := upgrade.NewHTTPPublisher("http://172.20.0.1:8000")

	log.Info().Msg("start upgrade daemon")

	if err := u.Run(time.Second*10, publisher); err != nil {
		log.Error().Err(err).Msg("fail to apply upgrade")
	}
}
