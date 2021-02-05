package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/threefoldtech/zos/pkg/network/dhcp"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/options"

	"github.com/containernetworking/plugins/pkg/ns"
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

// Requires tesll the analyser to wait for ip type
type Requires int

const (
	// RequiresIPv4 requires ipv4
	RequiresIPv4 Requires = 1 << iota
	// RequiresIPv6 requires ipv6
	RequiresIPv6
)

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

// AnalyseLinks is used to gather the IP that each interfaces would be from DHCP and SLAAC
// it returns the IPs and routes for each interfaces for both IPv4 and IPv6
//
// It will list all the physical interfaces that have a cable plugged in it
// create a network namespace per interfaces
// start a DHCP probe on each interfaces and gather the IPs and routes received
func AnalyseLinks(requires Requires, filters ...Filter) ([]IfaceConfig, error) {
	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return nil, err
	}

	filterred := links[:0]
filter:
	for _, link := range links {
		log := log.With().Str("interface", link.Attrs().Name).Str("type", link.Type()).Logger()

		log.Info().Msg("filtering interface")
		if link.Attrs().Name == "lo" {
			continue
		}

		for _, filter := range filters {
			ok, err := filter(link)
			if err != nil {
				return nil, errors.Wrap(err, "failed to filter link")
			}

			if !ok {
				log.Info().Msg("link didn't match filter crateria, skip testing")
				continue filter
			}
		}
		log.Info().Msg("link valid. testing link for connectivity")
		filterred = append(filterred, link)
	}

	wg := sync.WaitGroup{}

	ch := make(chan IfaceConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	for _, link := range filterred {
		wg.Add(1)
		analyseLink(ctx, &wg, ch, requires, link)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	configs := make([]IfaceConfig, 0, len(links))
	for cfg := range ch {
		log.Info().Str("link", cfg.Name).Str("info", fmt.Sprintf("%+v", cfg)).Msg("info")
		configs = append(configs, cfg)
	}

	return configs, nil
}

// AnalyseLink gets information about link
func AnalyseLink(ctx context.Context, requires Requires, link netlink.Link) (cfg IfaceConfig, err error) {
	cfg.Name = link.Attrs().Name
	if link.Attrs().MasterIndex != 0 {
		// this is to avoid breaking setups if a link is
		// already used
		return cfg, fmt.Errorf("link is attached to device")
	}
	tmpNS, err := namespace.Create(link.Attrs().Name)
	if err != nil {
		return cfg, errors.Wrap(err, "failed to create network namespace")
	}
	defer func() {
		if err := namespace.Delete(tmpNS); err != nil {
			log.Error().Err(err).Msgf("failed to delete network namespace %s", link.Attrs().Name)
		}
	}()

	if err := netlink.LinkSetNsFd(link, int(tmpNS.Fd())); err != nil {
		return cfg, errors.Wrap(err, "failed to sent interface into network namespace")
	}

	err = tmpNS.Do(func(host ns.NetNS) error {
		if err := netlink.LinkSetUp(link); err != nil {
			return errors.Wrap(err, "failed to bring interface up")
		}
		defer netlink.LinkSetDown(link)
		defer netlink.LinkSetNsFd(link, int(host.Fd()))

		name := link.Attrs().Name

		if err := options.Set(name, options.IPv6Disable(false)); err != nil {
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

	loop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
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

				for _, addr := range addrs {
					if addr.IP.IsGlobalUnicast() && !ifaceutil.IsULA(addr.IP) {
						addrs6.Add(addr)
					}
				}

				if ((requires&RequiresIPv6 == 0) || addrs6.Len() != 0) &&
					((requires&RequiresIPv4 == 0) || addrs4.Len() != 0) {
					break loop
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

		cfg = IfaceConfig{
			Name:    name,
			Addrs4:  addrs4.ToSlice(),
			Addrs6:  addrs6.ToSlice(),
			Routes4: routes4,
			Routes6: routes6,
		}

		return nil
	})

	return
}

func analyseLink(ctx context.Context, wg *sync.WaitGroup, out chan<- IfaceConfig, requires Requires, link netlink.Link) {
	go func() {
		defer wg.Done()
		cfg, err := AnalyseLink(ctx, requires, link)
		if err != nil {
			log.Error().Err(err).Str("interface", link.Attrs().Name).Msg("error analysing interface")
		}
		out <- cfg
	}()
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

// Filter interface to filter out links
type Filter func(link netlink.Link) (bool, error)

// PhysicalFilter returns true if physical link
func PhysicalFilter(link netlink.Link) (bool, error) {
	return link.Type() == "device", nil
}

// PluggedFilter returns true if link is plugged in
func PluggedFilter(link netlink.Link) (bool, error) {
	if err := netlink.LinkSetUp(link); err != nil {
		return false, errors.Wrapf(err, "failed to bring interface '%s' up", link.Attrs().Name)
	}

	return ifaceutil.IsVirtEth(link.Attrs().Name) || ifaceutil.IsPluggedTimeout(link.Attrs().Name, time.Second*10), nil
}

// NotAttachedFilter filters out network cards that
// area attached to bridges
func NotAttachedFilter(link netlink.Link) (bool, error) {
	return link.Attrs().MasterIndex == 0, nil
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

//TODO: re-implement as map
type addrSet map[string]netlink.Addr

func newAddrSet() addrSet {
	return addrSet{}
}

func (s addrSet) Add(addr netlink.Addr) {
	s[addr.String()] = addr
}

func (s addrSet) AddSlice(addrs []netlink.Addr) {
	for _, addr := range addrs {
		s.Add(addr)
	}
}

func (s addrSet) Len() int {
	return len(s)
}

func (s addrSet) ToSlice() []netlink.Addr {
	var results []netlink.Addr
	for _, l := range s {
		results = append(results, l)
	}

	return results
}
