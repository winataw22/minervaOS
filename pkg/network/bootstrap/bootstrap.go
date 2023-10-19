package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"sort"
	"strings"
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
	Name      string
	Addrs4    []netlink.Addr
	Addrs6    []netlink.Addr
	DefaultGW net.IP
}

// Requires tells the analyzer to wait for ip type
type Requires struct {
	ipv4 bool
	ipv6 bool
	vlan *uint16
}

var (
	// RequiresIPv4 requires ipv4
	RequiresIPv4 = Requires{ipv4: true}
	// RequiresIPv6 requires ipv6
	RequiresIPv6 = Requires{ipv6: true}
)

func (r Requires) WithIPv4(b bool) Requires {
	r.ipv4 = b
	return r
}

func (r Requires) WithIPv6(b bool) Requires {
	r.ipv6 = b
	return r
}

func (r Requires) WithVlan(b *uint16) Requires {
	r.vlan = b
	return r
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

func filterName(filter Filter) string {
	fn := runtime.FuncForPC(reflect.ValueOf(filter).Pointer())
	if fn == nil {
		return fmt.Sprintf("filter(%+v)", filter)
	}
	name := fn.Name()
	parts := strings.Split(name, ".")
	idx := len(parts) - 1
	if idx < 0 {
		return name
	}

	return parts[idx]
}

// AnalyzeLinks is used to gather the IP that each interfaces would be from DHCP and SLAAC
// it returns the IPs and routes for each interfaces for both IPv4 and IPv6
//
// It will list all the physical interfaces that have a cable plugged in it
// create a network namespace per interfaces
// start a DHCP probe on each interfaces and gather the IPs and routes received
func AnalyzeLinks(requires Requires, filters ...Filter) ([]IfaceConfig, error) {
	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return nil, err
	}

	filtered := links[:0]
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
				log.Info().Msgf("link didn't match filter criteria (%v), skip testing", filterName(filter))
				continue filter
			}
		}
		log.Info().Msg("link valid. testing link for connectivity")
		filtered = append(filtered, link)
	}

	wg := sync.WaitGroup{}

	ch := make(chan analyzeResult)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	for _, link := range filtered {
		wg.Add(1)
		analyzeLinkAsync(ctx, &wg, ch, requires, link)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	configs := make([]IfaceConfig, 0, len(links))
	for res := range ch {
		if res.err != nil {
			log.Error().Err(res.err).Str("interface", res.cfg.Name).Msg("error analyzing interface")
			continue
		}
		configs = append(configs, res.cfg)
	}

	return configs, nil
}

// analyzeLink gets information about link
func analyzeLink(ctx context.Context, requires Requires, link netlink.Link) (cfg IfaceConfig, err error) {
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
		defer func() {
			_ = netlink.LinkSetNsFd(link, int(host.Fd()))
			_ = netlink.LinkSetDown(link)
		}()

		name := link.Attrs().Name

		if err := options.Set(name, options.IPv6Disable(false)); err != nil {
			return errors.Wrapf(err, "failed to enable ip6 on %s", name)
		}

		cfg.Name = name

		if requires.vlan != nil {
			vlanID := *requires.vlan
			log.Debug().Uint32("vlan", uint32(vlanID)).Msg("setting up vlan interface for probing")
			name = fmt.Sprintf("%s.%d", name, vlanID)
			vl := netlink.Vlan{
				LinkAttrs: netlink.LinkAttrs{
					Name:        name,
					ParentIndex: link.Attrs().Index,
				},
				VlanId:       int(vlanID),
				VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
			}

			if err := netlink.LinkAdd(&vl); err != nil {
				return errors.Wrap(err, "failed to create vlan link for probing")
			}

			link, err = netlink.LinkByName(name)
			if err != nil {
				return errors.Wrap(err, "failed to get vlan link for probing")
			}

			if err := netlink.LinkSetUp(link); err != nil {
				return errors.Wrap(err, "failed to set up vlan link for probing")
			}

			defer func() {
				_ = netlink.LinkDel(link)
			}()
		}

		if err := options.Set(name, options.IPv6Disable(false)); err != nil {
			return errors.Wrapf(err, "failed to enable ip6 on %s", name)
		}

		log.Info().Str("interface", name).Msg("start DHCP probe")

		if requires.ipv4 {
			// requires IPv4
			probe, err := dhcp.Probe(ctx, name)
			if err != nil {
				return errors.Wrapf(err, "no ip v4 on interface '%s'", name)
			}
			ip, err := probe.IPNet()
			if err != nil {
				return errors.Wrap(err, "invalid ip address returned by dhcp")
			}
			cfg.Addrs4 = append(cfg.Addrs4, netlink.Addr{IPNet: ip})
			if len(probe.Router) != 0 {
				cfg.DefaultGW = net.ParseIP(probe.Router).To4()
			}
		}

		var addrs6 = newAddrSet()

	loop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
				if err != nil {
					return errors.Wrapf(err, "could not read address from interface %s", name)
				}

				for _, addr := range addrs {
					if addr.IP.IsGlobalUnicast() && !ifaceutil.IsULA(addr.IP) {
						addrs6.Add(addr)
					}
				}

				if !requires.ipv6 || addrs6.Len() != 0 {
					break loop
				}

				time.Sleep(time.Second)
			}
		}

		cfg.Addrs6 = addrs6.ToSlice()
		return nil
	})

	return
}

type analyzeResult struct {
	cfg IfaceConfig
	err error
}

func analyzeLinkAsync(ctx context.Context, wg *sync.WaitGroup, out chan<- analyzeResult, requires Requires, link netlink.Link) {
	go func() {
		defer wg.Done()
		cfg, err := analyzeLink(ctx, requires, link)
		out <- analyzeResult{cfg, err}
	}()
}

// SelectZOS decide which interface should be assigned to the ZOS bridge
// if multiple interfaces receives an IP from DHCP
// we prefer a interfaces that has the smallest IP and private IP gateway
// if none is found, then we pick the interface that has the smallest IP and any IP gateway
func SelectZOS(cfgs []IfaceConfig) (string, error) {

	selected4 := cfgs[:0]
	for _, cfg := range cfgs {
		if cfg.DefaultGW != nil && isPrivateIP(cfg.DefaultGW) {
			selected4 = append(selected4, cfg)
		}
	}

	if len(selected4) == 0 {
		for _, cfg := range cfgs {
			if cfg.DefaultGW != nil {
				selected4 = append(selected4, cfg)
			}
		}

	}
	if len(selected4) == 0 {
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

func VEthFilter(link netlink.Link) (bool, error) {
	return link.Type() == "veth", nil
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

func NotIpsAssignedFilter(link netlink.Link) (bool, error) {
	adds, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, err
	}

	return len(adds) == 0, nil
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

// TODO: re-implement as map
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
