package network

import (
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

// DHCPProbe will do a dhcp request on the interface inf
// if the interface gets a lease from the dhcp server, dhcpProbe return true and a nil error
// if something unexpected happens a non nil error is return
// if the interface didn't receive an lease, false and a nil error is returns
func dhcpProbe(inf string) (bool, error) {
	link, err := netlink.LinkByName(inf)
	if err != nil {
		return false, err
	}

	cmd := exec.Command("udhcpc",
		"-f", //foreground
		"-i", inf,
		"-t", "3", //try 3 times before giving up
		"-A", "3", //wait 3 seconds between each trial
		"-s", "/usr/share/udhcp/simple.script",
		"--now", // exit if lease is not obtained
	)

	if err := cmd.Start(); err != nil {
		return false, err
	}

	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Error().Err(err).Msg("")
		}

		_ = cmd.Wait()
	}()

	timeout := time.After(time.Second * 10)
	var (
		hasGW = false
		stay  = true
	)

	for !hasGW && stay {
		time.Sleep(time.Second)
		select {
		case <-timeout:
			stay = false
		default:
			for _, family := range []int{netlink.FAMILY_V6, netlink.FAMILY_V4} {
				hasGW, err = hasDefaultGW(link, family)
				if err != nil {
					return false, err
				}
				if hasGW {
					break
				}
			}
		}
	}

	return hasGW, nil
}

func hasDefaultGW(link netlink.Link, family int) (bool, error) {

	addrs, err := netlink.AddrList(link, family)
	if err != nil {
		return false, err
	}

	if len(addrs) <= 0 {
		return false, nil
	}

	log.Info().Msg("IP addresses found")
	for i, addr := range addrs {
		log.Info().
			Str("interface", link.Attrs().Name).
			IPAddr(string(i), addr.IP).Msg("")
	}

	routes, err := netlink.RouteList(link, family)
	if err != nil {
		return false, err
	}
	log.Info().Msg("routes found")
	for i, route := range routes {
		log.Info().
			Str("interface", link.Attrs().Name).
			Str(string(i), route.String())
	}

	for _, route := range routes {
		if route.Gw != nil {
			return true, err
		}
	}

	return false, nil
}
