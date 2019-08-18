package main

import (
	"os"
	"time"

	"github.com/threefoldtech/zosv2/modules/zinit"

	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/upgrade"
	"github.com/threefoldtech/zosv2/modules/version"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	zinitSocket = "/var/run/zinit.sock"
)

func main() {
	var (
		root     string
		broker   string
		url      string
		interval int
		ver      bool
	)

	flag.StringVar(&root, "root", "/var/modules/upgrade", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.StringVar(&url, "url", "https://versions.dev.grid.tf", "url of the upgrade server")
	flag.IntVar(&interval, "interval", 600, "interval in seconds between update checks, default to 600")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	zbusClient, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Error().Err(err).Msg("fail to connect to broker")
		return
	}

	flister := stubs.NewFlisterStub(zbusClient)
	zinit := zinit.New(zinitSocket)
	if err := zinit.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zinit")
	}

	u := upgrade.New(root, flister, zinit)

	// watcher := upgrade.NewPeriodicWatcher(10 * time.Second)
	publisher := upgrade.NewHTTPPublisher(url)

	log.Info().Msg("start upgrade daemon")

	// try to upgrade as soon as we boot, then check periodically
	_ = u.Upgrade(publisher)

	ticker := time.NewTicker(time.Second * time.Duration(interval))

	for range ticker.C {
		if err := u.Upgrade(publisher); err != nil {
			log.Error().Err(err).Msg("fail to apply upgrade")
		}
	}

}
