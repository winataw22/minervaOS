package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/threefoldtech/zos/pkg/version"
)

func main() {
	app.Initialize()

	var ver bool

	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if err := ifaceutil.SetLoUp(); err != nil {
		return
	}

	if err := configureZOS(); err != nil {
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
		cmd := exec.Command("wget", "google.com", "-O", "/dev/null", "-T", "5")
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

func configureZOS() error {
	f := func() error {
		z := zinit.Default()

		log.Info().Msg("Start network bootstrap")

		ifaceConfigs, err := bootstrap.AnalyzeLinks(
			bootstrap.RequiresIPv4,
			bootstrap.PhysicalFilter,
			bootstrap.PluggedFilter)
		if err != nil {
			log.Error().Err(err).Msg("failed to gather network interfaces configuration")
			return err
		}

		zosChild, err := bootstrap.SelectZOS(ifaceConfigs)
		if err != nil {
			log.Error().Err(err).Msg("failed to select a valid interface for zos bridge")
			return err
		}

		br, err := bootstrap.CreateDefaultBridge(types.DefaultBridge)
		if err != nil {
			return err
		}

		time.Sleep(time.Second) // this is dirty

		link, err := netlink.LinkByName(zosChild)
		if err != nil {
			return errors.Wrapf(err, "could not get link %s", zosChild)
		}

		log.Info().
			Str("device", link.Attrs().Name).
			Str("bridge", br.Name).
			Msg("attach interface to bridge")

		if err := bridge.AttachNicWithMac(link, br); err != nil {
			log.Error().Err(err).
				Str("device", link.Attrs().Name).
				Str("bridge", br.Name).
				Msg("fail to attach device to bridge")
			return err
		}

		if err := options.Set(zosChild, options.IPv6Disable(true)); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on zos slave %s", zosChild)
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return errors.Wrapf(err, "could not bring %s up", zosChild)
		}

		log.Info().Msg("writing udhcp init service")

		err = zinit.AddService("dhcp-zos", zinit.InitService{
			Exec:    fmt.Sprintf("/sbin/udhcpc -v -f -i %s -s /usr/share/udhcp/simple.script", types.DefaultBridge),
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

		if err := z.Start("dhcp-zos"); err != nil {
			log.Error().Err(err).Msg("fail to start dhcp-zos zinit service")
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
