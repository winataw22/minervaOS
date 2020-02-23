package bootstrap

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/threefoldtech/zos/pkg/network/dhcp"
	"github.com/threefoldtech/zos/pkg/network/namespace"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/vishvananda/netlink"
)

// IfaceConfig contains all the IP address and routes of an interface
type IfaceConfig struct {
	Name    string
	Addrs4  []netlink.Addr
	Addrs6  []netlink.Addr
	Routes4 []netlink.Route
	Routes6 []netlink.Route
}

type byIP4 []IfaceConfig

func (a byIP4) Len() int      { return len(a) }
func (a byIP4) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIP4) Less(i, j int) bool {
	sort.Sort(byAddr(a[i].Addrs4))
	sort.Sort(byAddr(a[j].Addrs4))
	return bytes.Compare(a[i].Addrs4[0].IP, a[j].Addrs4[0].IP) < 0
}

type byAddr []netlink.Addr

func (a byAddr) Len() int           { return len(a) }
func (a byAddr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAddr) Less(i, j int) bool { return bytes.Compare(a[i].IP, a[j].IP) < 0 }

// InspectIfaces is used to gather the IP that each interfaces would be from DHCP and SLAAC
// it returns the IPs and routes for each interfaces for both IPv4 and IPv6
//
// It will list all the physical interfaces that have a cable plugged in it
// create a network namespace per interfaces
// start a DHCP probe on each interfaces and gather the IPs and routes received
func InspectIfaces() ([]IfaceConfig, error) {
	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return nil, err
	}

	filters := []ifaceFilter{filterPhysical, filterPlugged}
	for _, filter := range filters {
		links = filter(links)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(links))

	cCfg := make(chan IfaceConfig)

	for _, link := range links {
		go func(cAddrs chan IfaceConfig, link netlink.Link) {
			defer wg.Done()
			if err := analyseLink(cAddrs, link); err != nil {
				log.Error().Err(err).Str("interface", link.Attrs().Name).Msg("error analysing interface")
			}
		}(cCfg, link)
	}

	go func() {
		wg.Wait()
		close(cCfg)
	}()

	configs := make([]IfaceConfig, 0, len(links))
	for cfg := range cCfg {
		configs = append(configs, cfg)
	}

	return configs, nil
}

func analyseLink(cAddrs chan IfaceConfig, link netlink.Link) error {
	tmpNS, err := namespace.Create(link.Attrs().Name)
	if err != nil {
		return errors.Wrap(err, "failed to create network namespace")
	}
	defer func() {
		if err := namespace.Delete(tmpNS); err != nil {
			log.Error().Err(err).Msgf("failed to delete network namespace %s", link.Attrs().Name)
		}
	}()

	if err := netlink.LinkSetNsFd(link, int(tmpNS.Fd())); err != nil {
		return errors.Wrap(err, "failed to sent interface into network namespace")
	}

	err = tmpNS.Do(func(_ ns.NetNS) error {
		if err := netlink.LinkSetUp(link); err != nil {
			return errors.Wrap(err, "failed to bring interface up")
		}
		defer netlink.LinkSetDown(link)

		name := link.Attrs().Name
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "0"); err != nil {
			return errors.Wrapf(err, "failed to enable ip6 on %s", name)
		}

		log.Info().Str("interface", name).Msg("start DHCP probe")

		probe := dhcp.NewProbe()
		if err := probe.Start(name); err != nil {
			return errors.Wrap(err, "error duging DHCP probe")
		}
		defer func() {
			if err := probe.Stop(); err != nil {
				log.Error().Err(err).Msg("could not stop DHCP probe properly")
			}
		}()

		var addrs4 = newAddrSet()
		var addrs6 = newAddrSet()
		cTimeout := time.After(time.Second * 122)

	Loop:
		for {
			select {
			case <-cTimeout:
				break Loop
			default:
				addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
				if err != nil {
					return errors.Wrapf(err, "could not read address from interface %s", name)
				}
				addrs4.AddSlice(addrs)

				addrs, err = netlink.AddrList(link, netlink.FAMILY_V6)
				if err != nil {
					return errors.Wrapf(err, "could not read address from interface %s", name)
				}
				addrs6.AddSlice(addrs)

				if addrs6.Len() > 0 && addrs4.Len() > 0 {
					break Loop
				}
				time.Sleep(time.Second)
			}
		}

		routes4, err := netlink.RouteList(link, netlink.FAMILY_V4)
		if err != nil {
			return errors.Wrapf(err, "could not read routes from interface %s", name)
		}
		routes6, err := netlink.RouteList(link, netlink.FAMILY_V6)
		if err != nil {
			return errors.Wrapf(err, "could not read routes from interface %s", name)
		}

		cAddrs <- IfaceConfig{
			Name:    name,
			Addrs4:  addrs4.ToSlice(),
			Addrs6:  addrs6.ToSlice(),
			Routes4: routes4,
			Routes6: routes6,
		}

		return nil
	})
	if err != nil {
		log.Info().Str("interface", link.Attrs().Name).Msg("failed to gather ip addresses")
	}

	return nil
}

// SelectZOS decide which interface should be assigned to the ZOS bridge
// if multiple interfaces receives an IP from DHCP
// we prefer a interfaces that has the smallest IP and private IP gateway
// if none is found, then we pick the interface that has the smallest IP and any IP gateway
func SelectZOS(cfgs []IfaceConfig) (string, error) {

	selected4 := cfgs[:0]
	// selected6 := cfgs[:0]
	for _, cfg := range cfgs {
		for _, route := range cfg.Routes4 {
			if route.Gw != nil && isPrivateIP(route.Gw) {
				selected4 = append(selected4, cfg)
			}
		}

		// for _, route := range cfg.Routes4 {
		// 	if route.Gw != nil && isPrivateIP(route.Gw) {
		// 		selected6 = append(selected6, cfg)
		// 	}
		// }
	}

	if len(selected4) < 1 {
		for _, cfg := range cfgs {
			for _, route := range cfg.Routes4 {
				if route.Gw != nil {
					selected4 = append(selected4, cfg)
				}
			}

			// for _, route := range cfg.Routes6 {
			// 	if route.Gw != nil {
			// 		selected6 = append(selected6, cfg)
			// 	}
			// }
		}
	}

	if len(selected4) < 1 {
		return "", fmt.Errorf("no route with default gateway found")
	}

	sort.Sort(byIP4(selected4))

	return selected4[0].Name, nil
}

type ifaceFilter func(links []netlink.Link) []netlink.Link

// filterPhysical is a ifaceFilter that filter out the links that are
// not physical interface
func filterPhysical(links []netlink.Link) []netlink.Link {
	out := links[:0]
	for _, l := range links {
		if l.Type() == "device" {
			out = append(out, l)
		}
	}
	return out
}

// filterPlugged is a ifaceFilter that filter out the links that does not have a cable plugged in it
func filterPlugged(links []netlink.Link) []netlink.Link {
	out := links[:0]
	for _, l := range links {

		if err := netlink.LinkSetUp(l); err != nil {
			log.Info().Str("interface", l.Attrs().Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(l.Attrs().Name) && !ifaceutil.IsPluggedTimeout(l.Attrs().Name, time.Second*5) {
			log.Info().Str("interface", l.Attrs().Name).Msg("interface is not plugged in, skipping")
			continue
		}

		out = append(out, l)
	}
	return out
}

func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{}
	for _, cidr := range []string{
		// "127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

type addrSet struct {
	l []netlink.Addr
}

func newAddrSet() *addrSet {
	return &addrSet{
		l: make([]netlink.Addr, 0, 5),
	}
}

func (s *addrSet) Add(addr netlink.Addr) {
	for _, a := range s.l {
		if a.Equal(addr) {
			return
		}
	}
	s.l = append(s.l, addr)
}

func (s *addrSet) AddSlice(addrs []netlink.Addr) {
	for _, addr := range addrs {
		s.Add(addr)
	}
}

func (s *addrSet) Len() int {
	return len(s.l)
}
func (s *addrSet) ToSlice() []netlink.Addr {
	return s.l
}
