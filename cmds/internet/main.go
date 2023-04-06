package main

import (
	"flag"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/dhcp"
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

func debugZos() error {
	br, err := bridge.Get(types.DefaultBridge)
	if err != nil {
		return errors.Wrap(err, "failed to get default bridge")
	}

	addrs, err := netlink.AddrList(br, netlink.FAMILY_ALL)
	if err != nil {
		return errors.Wrap(err, "failed to list bridge addresses")
	}

	for _, addr := range addrs {
		log.Info().Str("address", addr.String()).Msg("default bridge has address")
	}

	routes, err := netlink.RouteList(br, netlink.FAMILY_ALL)
	if err != nil {
		return errors.Wrap(err, "failed to list bridge routes")
	}

	for _, route := range routes {
		log.Info().Str("route", route.String()).Msg("default bridge has routes")
	}

	return nil
}

func check() error {
	retries := 0
	f := func() error {
		retries += 1

		log.Info().Msg("testing internet connection. trying out bootstrap.grid.tf:80")
		// print some helpful debugging information to help with debugging issues
		// if internet connection wasn't established
		if err := debugZos(); err != nil {
			log.Error().Err(err).Msg("failed to list default bridge debug information")
		}

		// we only care about possibility of establishing a connection
		// so just establishing a connection then close it is good enough
		con, err := net.Dial("tcp", "bootstrap.grid.tf:http")
		if err != nil {
			return errors.Wrap(err, "failed to reach bootstrap.grid.tf")
		}
		return con.Close()
	}

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Info().Msg("internet connection is not ready yet")
			if retries%10 == 0 {
				log.Error().Err(err).Msgf("error while trying to test internet connectivity. %d retries attempted", retries)
			}
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}

func configureZOS() error {
	f := func() error {
		log.Info().Msg("Start network bootstrap")

		ifaceConfigs, err := bootstrap.AnalyzeLinks(
			bootstrap.RequiresIPv4,
			bootstrap.PhysicalFilter,
			bootstrap.PluggedFilter)
		if err != nil {
			log.Error().Err(err).Msg("failed to gather network interfaces configuration")
			return err
		}

		log.Info().Int("count", len(ifaceConfigs)).Msg("found interfaces with internet access")
		log.Info().Msgf("found interfaces: %+v", ifaceConfigs)
		zosChild, err := bootstrap.SelectZOS(ifaceConfigs)
		if err != nil {
			log.Error().Err(err).Msg("failed to select a valid interface for zos bridge")
			return err
		}

		log.Info().Str("interface", zosChild).Msg("selecting interface")
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

		dhcpService := dhcp.NewService(types.DefaultBridge, "", zinit.Default())
		if err := dhcpService.DestroyOlderService(); err != nil {
			log.Error().Err(err).Msgf("failed to destory older %s service", dhcpService.Name)
			return err
		}
		// create the new service anyway
		if err := dhcpService.Create(); err != nil {
			log.Error().Err(err).Msgf("failed to create %s service", dhcpService.Name)
			return err
		}

		if err := dhcpService.Start(); err != nil {
			log.Error().Err(err).Msgf("failed to start %s service", dhcpService.Name)
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
