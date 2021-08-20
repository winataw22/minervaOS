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
	"github.com/pkg/errors"

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
	log.Info().Str("iface", name).Msg("check if interface has a cable plugged in")
	c := time.After(duration)
	plugged := false
	for !plugged {
		select {
		case <-c:
			return false // timeout
		default:
			plugged = IsPlugged(name)
			if plugged {
				return true
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
func HasDefaultGW(link netlink.Link, family int) (bool, net.IP, error) {

	addrs, err := netlink.AddrList(link, family)
	if err != nil {
		return false, nil, err
	}

	if len(addrs) <= 0 {
		return false, nil, nil
	}

	routes, err := netlink.RouteList(link, family)
	if err != nil {
		return false, nil, err
	}

	for i, route := range routes {
		log.Info().
			Str("interface", link.Attrs().Name).
			Str(fmt.Sprint(i), route.String())
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
func MakeVethPair(name, master string, mtu int) (netlink.Link, error) {
	masterLink, err := netlink.LinkByName(master)
	if err != nil {
		return nil, err
	}

	peer := fmt.Sprintf("%s-p", name)
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  name,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: peer,
	}

	if err = netlink.LinkAdd(veth); err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = netlink.LinkDel(veth)
		}
	}()

	peerLink, err := netlink.LinkByName(peer)
	if err = netlink.LinkSetMaster(peerLink, masterLink); err != nil {
		return nil, err
	}

	// make sure the lowerhalf is up, this automatically sets the upperhalf UP
	if err = netlink.LinkSetUp(peerLink); err != nil {
		return nil, errors.Wrap(err, "could not set veth peer up")
	}

	// Re-fetch the link to get its creation-time parameters, e.g. index and mac
	veth2, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	return veth2, nil
}

// VethByName loads one end of a veth pair given its name
func VethByName(name string) (netlink.Link, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, errors.Wrapf(err, "veth %s not found", name)
	}

	if link.Type() != "veth" {
		return nil, fmt.Errorf("device '%s' is not a veth pair", name)
	}

	return link.(*netlink.Veth), nil

}

// Exists test check if the named interface exists
// if netNS is not nil switch in the network namespace
// before checking
func Exists(name string, netNS ns.NetNS) bool {
	exist := false
	if netNS != nil {
		err := netNS.Do(func(_ ns.NetNS) error {
			_, err := netlink.LinkByName(name)
			return err
		})
		exist = err == nil
	} else {
		_, err := netlink.LinkByName(name)
		exist = err == nil
	}
	return exist
}

// Get link by name from optional namespace
func Get(name string, netNS ns.NetNS) (link netlink.Link, err error) {
	if netNS != nil {
		err = netNS.Do(func(_ ns.NetNS) error {
			link, err = netlink.LinkByName(name)
			return err
		})

		return
	}

	link, err = netlink.LinkByName(name)
	return
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
	f := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(name)
		if err != nil {
			return err
		}

		// early return if the mac address if already properly set
		actualMac := link.Attrs().HardwareAddr
		if bytes.Equal(actualMac, mac) {
			return nil
		}

		if err := netlink.LinkSetDown(link); err != nil {
			return err
		}
		defer func() {
			_ = netlink.LinkSetUp(link)
		}()

		return netlink.LinkSetHardwareAddr(link, mac)
	}

	if netNS != nil {
		return netNS.Do(f)
	}
	return f(nil)
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

// HostIPV6Iface return the first physical interface to have an
// ipv6 public address
func HostIPV6Iface(useZos bool) (string, error) {

	links, err := netlink.LinkList()
	if err != nil {
		return "", err
	}
	zos, err := netlink.LinkByName("zos")
	if err != nil {
		return "", err
	}

	// first check all physical interface
	links = LinkFilter(links, []string{"device"})
	// then check zos bridge
	if useZos {
		links = append(links, zos)
	}

	for _, link := range links {

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			log.Info().
				Str("iface", link.Attrs().Name).
				Str("addr", addr.String()).
				Msg("search public ipv6 address")

			if addr.IP.IsGlobalUnicast() && !IsULA(addr.IP) {
				return link.Attrs().Name, nil
			}
		}
	}

	return "", fmt.Errorf("no valid IPv6 address found in host namespace")
}

// ParentIface return the parent interface fof iface
// if netNS is not nil, switch to the network namespace before checking iface
func ParentIface(iface string, netNS ns.NetNS) (netlink.Link, error) {
	var (
		parentIndex int
		err         error
	)

	f := func(_ ns.NetNS) error {
		master, err := netlink.LinkByName(iface)
		if err != nil {
			return err
		}

		parentIndex = master.Attrs().ParentIndex
		return nil
	}

	if netNS != nil {
		err = netNS.Do(f)
	} else {
		err = f(nil)
	}
	if err != nil {
		return nil, err
	}

	return netlink.LinkByIndex(parentIndex)
}

var ulaPrefix = net.IPNet{
	IP:   net.ParseIP("fc00::"),
	Mask: net.CIDRMask(7, 128),
}

// IsULA checks if IPv6 is a ULA ip
func IsULA(ip net.IP) bool {
	return ulaPrefix.Contains(ip)
}
