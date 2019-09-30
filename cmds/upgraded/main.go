package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/threefoldtech/zosv2/modules/utils"
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

// setup is a sanity check function, the whole purpose of this
// is to make sure at least required services are running in case
// of upgrade failure
// for example, in case of upgraded crash after it already stopped all
// the services for upgrade.
func setup(zinit *zinit.Client) error {
	for _, required := range []string{"redis", "flistd"} {
		if err := zinit.StartWait(5*time.Second, required); err != nil {
			return err
		}
	}

	return nil
}

// SafeUpgrade makes sure upgrade daemon is not interrupted
// While
func SafeUpgrade(upgrader *upgrade.Upgrader) error {
	ch := make(chan os.Signal)
	defer close(ch)
	defer signal.Stop(ch)

	// try to upgraded to latest
	// but mean while also make sure the daemon can not be killed by a signal
	signal.Notify(ch)
	return upgrader.Upgrade()
}

func main() {
	var (
		root     string
		broker   string
		interval int
		ver      bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/upgraded", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.IntVar(&interval, "interval", 600, "interval in seconds between update checks, default to 600")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if upgrade.DetectBootMethod() != upgrade.BootMethodFList {
		log.Info().Msg("not booted with an flist. life upgrade is not supported")
		// wait forever
		select {}
	}

	zinit, err := zinit.New(zinitSocket)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zinit")
	}

	// recover procedure to make sure upgrade always has what it needs
	// to work
	if err := setup(zinit); err != nil {
		log.Fatal().Err(err).Msg("upgraded setup failed")
	}

	zbusClient, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Error().Err(err).Msg("fail to connect to broker")
		return
	}

	flister := stubs.NewFlisterStub(zbusClient)

	upgrader := upgrade.Upgrader{
		FLister: flister,
		Zinit:   zinit,
	}

	log.Info().Msg("start upgrade daemon")

	ticker := time.NewTicker(time.Second * time.Duration(interval))

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	for {
		err := SafeUpgrade(&upgrader)
		if err == upgrade.ErrRestartNeeded {
			log.Info().Msg("restarting upgraded")
			return
		} else if err != nil {
			//TODO: crash or continue!
			log.Error().Err(err).Msg("upgrade failed")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			break
		}
	}
}
