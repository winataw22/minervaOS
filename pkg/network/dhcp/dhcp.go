package dhcp

import (
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/vishvananda/netlink"
)

// Probe will do a dhcp request on the interface inf
// if the interface gets a lease from the dhcp server, dhcpProbe return true and a nil error
// if something unexpected happens a non nil error is return
// if the interface didn't receive an lease, false and a nil error is returns
func Probe(inf string) (bool, error) {
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
			log.Error().Err(err).Send()
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
			hasGW, _, err = ifaceutil.HasDefaultGW(link)
			if err != nil {
				return false, err
			}
			if hasGW {
				break
			}

		}
	}

	return hasGW, nil
}
