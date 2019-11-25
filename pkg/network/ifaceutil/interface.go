package ifaceutil

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"

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
	for _, addr := range addrs {
		log.Info().
			Str("interface", link.Attrs().Name).
			IPAddr("ip", addr.IP).Send()
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

// RandomName generate a random string that can be used for
// interface or network namespace
// if prefix is not None, the random name is prefixed with it
func RandomName(prefix string) (string, error) {
	b := make([]byte, 4)
	_, err := rand.Reader.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random name: %v", err)
	}
	return fmt.Sprintf("%s%x", prefix, b), nil
}

// MakeVethPair creates a veth pair
func MakeVethPair(name, peer string, mtu int) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  name,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: peer,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}
	// Re-fetch the link to get its creation-time parameters, e.g. index and mac
	veth2, err := netlink.LinkByName(name)
	if err != nil {
		netlink.LinkDel(veth) // try and clean up the link if possible.
		return nil, err
	}

	return veth2, nil
}

// Exists test check if the named interface exists
// if netNS is not nil switch in the network namespace
// before checking
func Exists(name string, netNS ns.NetNS) bool {
	exist := false
	if netNS != nil {
		netNS.Do(func(_ ns.NetNS) error {
			_, err := netlink.LinkByName(name)
			exist = err == nil
			return nil
		})
	} else {
		_, err := netlink.LinkByName(name)
		exist = err == nil
	}
	return exist
}

// GetMAC gets the mac address from the Interface
func GetMAC(name string, netNS ns.NetNS) (net.HardwareAddr, error) {
	if netNS != nil {
		var mac net.HardwareAddr
		err := netNS.Do(func(_ ns.NetNS) error {
			link, err := netlink.LinkByName(name)
			if err != nil {
				return err
			}
			mac = link.Attrs().HardwareAddr
			return nil
		})
		return mac, err
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}
	return link.Attrs().HardwareAddr, nil
}

// SetMAC Sets the mac addr of an interface
// if netNS is not nil switch in the network namespace
// before setting
func SetMAC(name string, mac net.HardwareAddr, netNS ns.NetNS) error {
	if netNS != nil {
		return netNS.Do(func(_ ns.NetNS) error {
			link, err := netlink.LinkByName(name)
			if err != nil {
				return err
			}
			if err := netlink.LinkSetDown(link); err != nil {
				return err
			}
			defer netlink.LinkSetUp(link)

			return netlink.LinkSetHardwareAddr(link, mac)

		})
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	if err := netlink.LinkSetDown(link); err != nil {
		return err
	}
	defer netlink.LinkSetUp(link)
	return netlink.LinkSetHardwareAddr(link, mac)
}

// Delete deletes the named interface
// if netNS is not nil Exists switch in the network namespace
// before deleting
func Delete(name string, netNS ns.NetNS) error {
	if netNS != nil {
		return netNS.Do(func(_ ns.NetNS) error {
			link, err := netlink.LinkByName(name)
			if err != nil {
				if !os.IsNotExist(err) {
					return nil
				}
				return err
			}
			return netlink.LinkDel(link)
		})
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return netlink.LinkDel(link)
}
