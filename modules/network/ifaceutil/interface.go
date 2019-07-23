package ifaceutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
)

const carrierFile = "/sys/class/net/%s/carrier"

// LinkFilter list all the links of a certain type
func LinkFilter(links []netlink.Link, types []string) []netlink.Link {
	out := make([]netlink.Link, 0, len(links))
	for _, link := range links {
		for _, t := range types {
			if link.Type() == t {
				out = append(out, link)
				break
			}
		}
	}
	return out
}

// IsPlugged test if an interface has a cable plugged in
func IsPlugged(inf string) bool {
	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
	if err != nil {
		return false
	}
	data = bytes.TrimSpace(data)
	return string(data) == "1"
}

// IsPluggedTimeout is like IsPlugged but retry for duration time before returning
func IsPluggedTimeout(name string, duration time.Duration) bool {
	plugged := false
	c := time.After(duration)
	for out := false; out == false; {
		select {
		case <-c:
			out = true
			break
		default:
			plugged = IsPlugged(name)
			if plugged {
				out = true
				break
			}
		}
		time.Sleep(time.Second)
	}
	return plugged
}

// IsVirtEth tests if an interface is a veth
func IsVirtEth(inf string) bool {
	path := fmt.Sprintf("/sys/class/net/%s/device", inf)
	dest, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return strings.Contains(filepath.Base(dest), "virtio")
}

// HasDefaultGW tests if a link as a default gateway configured
// it return the ip of the gateway if there is one
func HasDefaultGW(link netlink.Link) (bool, net.IP, error) {

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, nil, err
	}

	if len(addrs) <= 0 {
		return false, nil, nil
	}

	log.Info().Msg("IP addresses found")
	for i, addr := range addrs {
		log.Info().
			Str("interface", link.Attrs().Name).
			IPAddr(string(i), addr.IP).Msg("")
	}

	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, nil, err
	}
	log.Info().Msg("routes found")
	for i, route := range routes {
		log.Info().
			Str("interface", link.Attrs().Name).
			Str(string(i), route.String())
	}

	for _, route := range routes {
		if route.Gw != nil {
			return true, route.Gw, err
		}
	}

	return false, nil, nil
}

// SetLoUp brings the lo interface up
func SetLoUp() error {
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		log.Error().Err(err).Msg("fail to get lo interface")
		return err
	}
	if err := netlink.LinkSetUp(lo); err != nil {
		log.Error().Err(err).Msg("fail to bring lo interface up")
		return err
	}
	return err
}
