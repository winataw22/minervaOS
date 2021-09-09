package public

import (
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	toZosVeth                   = "tozos" // veth pair from br-pub to zos
	publicNsMACDerivationSuffix = "-public"

	// PublicBridge public bridge name, exists only after a call to EnsurePublicSetup
	PublicBridge    = types.PublicBridge
	PublicNamespace = types.PublicNamespace
)

// EnsurePublicBridge makes sure that the public bridge exists
func ensurePublicBridge() (*netlink.Bridge, error) {
	br, err := bridge.Get(PublicBridge)
	if err != nil {
		br, err = bridge.New(PublicBridge)
		if err != nil {
			return nil, err
		}
	}

	if err := options.Set(
		br.Attrs().Name,
		options.IPv6Disable(true),
		options.AcceptRA(options.RAOff)); err != nil {
		return nil, err
	}

	return br, nil
}

//getPublicNamespace gets the public namespace, or nil if it's
//not setup or does not exist. the caller must be able to handle
//this case
func getPublicNamespace() ns.NetNS {
	ns, _ := namespace.GetByName(PublicNamespace)
	return ns
}

// IPs gets the public ips of the nodes
func IPs() ([]net.IPNet, error) {
	namespace := getPublicNamespace()
	if namespace == nil {
		return nil, nil
	}

	defer namespace.Close()

	var ips []net.IPNet
	err := namespace.Do(func(_ ns.NetNS) error {
		ln, err := netlink.LinkByName(types.PublicIface)
		if err != nil {
			return errors.Wrap(err, "failed to get public interface")
		}

		results, err := netlink.AddrList(ln, netlink.FAMILY_ALL)
		if err != nil {
			return errors.Wrap(err, "failed to list ips for public interface")
		}

		for _, ip := range results {
			ips = append(ips, *ip.IPNet)
		}
		return nil
	})

	return ips, err
}

func setupPublicBridge(br *netlink.Bridge) error {
	// find possible exit interface.
	// or fall back to zos.
	exit, err := findPossibleExit()
	if err != nil {
		return errors.Wrap(err, "failed to find possible exit")
	}

	log.Debug().Str("master", exit).Msg("public master (exit)")
	exitLink, err := netlink.LinkByName(exit)
	if err != nil {
		return errors.Wrapf(err, "failed to get link '%s' by name", exit)
	}

	if err := netlink.LinkSetUp(exitLink); err != nil {
		return errors.Wrapf(err, "failed to set link '%s' up", exitLink.Attrs().Name)
	}

	if err := bridge.Attach(exitLink, br, toZosVeth); err != nil {
		return errors.Wrap(err, "failed to attach exit nic to public bridge 'br-pub'")
	}

	return nil
}

func HasPublicSetup() bool {
	return namespace.Exists(PublicNamespace)
}

// GetPublicSetup gets the public setup from reality
// or error if node has no public setup
func GetPublicSetup() (pkg.PublicConfig, error) {
	if !namespace.Exists(PublicNamespace) {
		return pkg.PublicConfig{}, ErrNoPublicConfig
	}

	namespace, err := namespace.GetByName(PublicNamespace)
	if err != nil {
		return pkg.PublicConfig{}, err
	}
	var cfg pkg.PublicConfig
	if set, err := LoadPublicConfig(); err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "failed to load configuration")
	} else {
		// we only need the domain name from the config
		cfg.Domain = set.Domain
	}
	// everything else is loaded from the actual state of the node.
	err = namespace.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(types.PublicIface)
		if err != nil {
			return errors.Wrap(err, "failed to get public interface")
		}

		ips, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return errors.Wrap(err, "failed to get public ipv4")
		}
		if len(ips) > 0 {
			cfg.IPv4 = gridtypes.IPNet{IPNet: *ips[0].IPNet}
		}
		routes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, nil, 0)
		if err != nil {
			return errors.Wrap(err, "failed to get ipv4 default gateway")
		}
		for _, r := range routes {
			if r.Dst == nil {
				cfg.GW4 = r.Gw
				break
			}
		}

		ips, err = netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return errors.Wrap(err, "failed to get public ipv4")
		}

		for _, ip := range ips {
			if ip.IP.IsGlobalUnicast() && !ifaceutil.IsULA(ip.IP) {
				cfg.IPv6 = gridtypes.IPNet{IPNet: *ips[0].IPNet}
			}
		}

		routes, err = netlink.RouteListFiltered(netlink.FAMILY_V6, nil, 0)
		if err != nil {
			return errors.Wrap(err, "failed to get ipv4 default gateway")
		}
		for _, r := range routes {
			if r.Dst == nil {
				cfg.GW6 = r.Gw
				break
			}
		}

		return nil
	})

	return cfg, err
}

// EnsurePublicSetup create the public setup, it's okay to have inf == nil
func EnsurePublicSetup(nodeID pkg.Identifier, inf *pkg.PublicConfig) (*netlink.Bridge, error) {
	log.Debug().Msg("ensure public setup")
	br, err := ensurePublicBridge()
	if err != nil {
		return nil, err
	}

	// find if we have anything attached to this bridge (so initialization)
	// has been done before.
	all, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list host links")
	}

	filtered := all[:0]
	for _, link := range all {
		if link.Attrs().MasterIndex == br.Index {
			filtered = append(filtered, link)
		}
	}

	if len(filtered) == 0 {
		if err := setupPublicBridge(br); err != nil {
			return nil, err
		}
	}

	if inf != nil {
		if err := setupPublicNS(nodeID, inf); err != nil {
			return nil, errors.Wrap(err, "failed to ensure public namespace setup")
		}
	}

	return br, netlink.LinkSetUp(br)
}

func findPossibleExit() (string, error) {
	log.Debug().Msg("find possible ipv6 exit interface")

	links, err := bootstrap.AnalyseLinks(
		bootstrap.RequiresIPv6,
		bootstrap.PhysicalFilter,
		bootstrap.NotAttachedFilter,
		bootstrap.PluggedFilter,
	)

	log.Debug().Int("found", len(links)).Msg("found possible links")
	if err != nil {
		return "", errors.Wrap(err, "failed to analyse links")
	}

	for _, link := range links {
		for _, addr := range link.Addrs6 {
			log.Debug().Str("link", link.Name).IPAddr("ip", addr.IP).Msg("checking address")
			if addr.IP.IsGlobalUnicast() && !ifaceutil.IsULA(addr.IP) {
				return link.Name, nil
			}
		}
	}

	return types.DefaultBridge, nil
}

func ensureNamespace() (ns.NetNS, error) {
	if !namespace.Exists(PublicNamespace) {
		log.Info().Str("namespace", PublicNamespace).Msg("Create network namespace")
		return namespace.Create(PublicNamespace)
	}

	return namespace.GetByName(PublicNamespace)
}

func ensurePublicMacvlan(iface *pkg.PublicConfig, pubNS ns.NetNS) (*netlink.Macvlan, error) {
	var (
		pubIface *netlink.Macvlan
		err      error
	)

	if !ifaceutil.Exists(types.PublicIface, pubNS) {

		switch iface.Type {
		case "":
			fallthrough
		case pkg.MacVlanIface:
			pubIface, err = macvlan.Create(types.PublicIface, types.PublicBridge, pubNS)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create public mac vlan interface")
			}
		default:
			return nil, fmt.Errorf("unsupported public interface type %s", iface.Type)
		}

	} else {
		err := pubNS.Do(func(_ ns.NetNS) error {
			pubIface, err = macvlan.GetByName(types.PublicIface)
			return err
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get public macvlan interface")
		}
	}

	return pubIface, nil
}

func publicConfig(iface *pkg.PublicConfig) (ips []*net.IPNet, routes []*netlink.Route, err error) {
	if !iface.IPv6.Nil() && iface.GW6 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw: iface.GW6,
		})
		ips = append(ips, &iface.IPv6.IPNet)
	}

	if !iface.IPv4.Nil() && iface.GW4 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw: iface.GW4,
		})
		ips = append(ips, &iface.IPv4.IPNet)
	}

	if len(ips) <= 0 || len(routes) <= 0 {
		err := fmt.Errorf("missing some information in the exit iface object")
		log.Error().Err(err).Msg("failed to configure public interface")
		return nil, nil, err
	}

	return ips, routes, nil
}

// setupPublicNS creates a public namespace in a node
func setupPublicNS(nodeID pkg.Identifier, iface *pkg.PublicConfig) error {
	pubNS, err := ensureNamespace()
	if err != nil {
		return err
	}

	defer pubNS.Close()

	pubIface, err := ensurePublicMacvlan(iface, pubNS)
	if err != nil {
		return err
	}

	log.Info().
		Str("pub iface", fmt.Sprintf("%+v", pubIface)).
		Msg("configure public interface inside public namespace")

	ips, routes, err := publicConfig(iface)
	if err != nil {
		return errors.Wrap(err, "failed configure public network interface")
	}

	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + publicNsMACDerivationSuffix))
	if err := macvlan.Install(pubIface, mac, ips, routes, pubNS); err != nil {
		return err
	}

	err = pubNS.Do(func(_ ns.NetNS) error {
		lo, err := netlink.LinkByName("lo")
		if err != nil {
			return errors.Wrap(err, "failed to get lo interface")
		}

		if err := netlink.LinkSetUp(lo); err != nil {
			return errors.Wrap(err, "failed to set lo interface up")
		}

		if err := options.SetIPv6AcceptRA(options.RAAcceptIfForwardingIsEnabled); err != nil {
			return errors.Wrap(err, "failed to accept_ra=2 in public namespace")
		}

		if err := options.SetIPv6LearnDefaultRouteInRA(true); err != nil {
			return errors.Wrap(err, "failed to enable enable_defrtr=1 in public namespace")
		}

		if err := options.SetIPv6Forwarding(true); err != nil {
			return errors.Wrap(err, "failed to enable ipv6 forwarding in public namespace")
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "error while configuring IPv6 public namespace")
	}

	return nil
}
