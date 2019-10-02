package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"

	"github.com/threefoldtech/zos/pkg/version"
	"github.com/threefoldtech/zos/pkg/zinit"
)

func main() {
	var (
		ver bool
	)

	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := ifaceutil.SetLoUp(); err != nil {
		return
	}

	if err := bootstrap(); err != nil {
		log.Error().Err(err).Msg("failed to bootstrap network")
		os.Exit(1)
	}

	// wait for internet connection
	if err := check(); err != nil {
		log.Error().Err(err).Msg("failed to check internet connection")
		os.Exit(1)
	}

	log.Info().Msg("network bootstrapped successfully")
}

func check() error {
	f := func() error {

		cmd := exec.Command("ping", "-c", "1", "google.com")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to test internet connectivity")
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}

func bootstrap() error {
	f := func() error {

		z, err := zinit.New("")
		if err != nil {
			log.Error().Err(err).Msg("failed to connect to zinit")
			return err
		}

		log.Info().Msg("Start network bootstrap")
		if err := network.Bootstrap(); err != nil {
			log.Error().Err(err).Msg("fail to boostrap network")
			return err
		}

		log.Info().Msg("writing udhcp init service")

		err = zinit.AddService("dhcp-zos", zinit.InitService{
			Exec:    fmt.Sprintf("/sbin/udhcpc -v -f -i %s -s /usr/share/udhcp/simple.script", network.DefaultBridge),
			Oneshot: false,
			After:   []string{},
		})

		if err != nil {
			log.Error().Err(err).Msg("fail to create dhcp-zos zinit service")
			return err
		}

		if err := z.Monitor("dhcp-zos"); err != nil {
			log.Error().Err(err).Msg("fail to start monitoring dhcp-zos zinit service")
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
