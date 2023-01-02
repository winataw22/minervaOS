package public

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

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
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/vishvananda/netlink"
)

const (
	toZosVeth                   = "tozos" // veth pair from br-pub to zos
	publicNsMACDerivationSuffix = "-public"

	// PublicBridge public bridge name, exists only after a call to EnsurePublicSetup
	PublicBridge    = types.PublicBridge
	PublicNamespace = types.PublicNamespace

	defaultPublicResolveConf = `nameserver 8.8.8.8
nameserver 1.1.1.1
nameserver 2001:4860:4860::8888
`
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

// getPublicNamespace gets the public namespace, or nil if it's
// not setup or does not exist. the caller must be able to handle
// this case
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
	exit, err := detectExitNic()
	if err != nil {
		return errors.Wrap(err, "failed to find possible exit")
	}

	log.Debug().Str("master", exit).Msg("public master (exit)")
	exitLink, err := netlink.LinkByName(exit)
	if err != nil {
		return errors.Wrapf(err, "failed to get link '%s' by name", exit)
	}

	return attachPublicToExit(br, exitLink)
}

func attachPublicToExit(br *netlink.Bridge, exit netlink.Link) error {
	if err := netlink.LinkSetUp(exit); err != nil {
		return errors.Wrapf(err, "failed to set link '%s' up", exit.Attrs().Name)
	}

	if err := bridge.Attach(exit, br, toZosVeth); err != nil {
		return errors.Wrap(err, "failed to attach exit nic to public bridge 'br-pub'")
	}

	return nil
}

func GetCurrentPublicExitLink() (netlink.Link, error) {
	// return the upstream (exit) link for br-pub
	br, err := bridge.Get(PublicBridge)
	if err != nil {
		return nil, errors.Wrap(err, "no public bridge found")
	}

	all, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list node nics")
	}

	// public bridge can be wired to either
	matches := []bootstrap.Filter{
		// a nic
		bootstrap.PhysicalFilter,
		// a veth pair to another bridge (zos always)
		bootstrap.VEthFilter,
	}

	for _, link := range all {
		for _, match := range matches {
			if ok, _ := match(link); !ok {
				continue
			}

			if link.Attrs().MasterIndex == br.Index {
				return link, nil
			}
		}
	}

	return nil, os.ErrNotExist
}

// SetPublicExitLink rewires the br-pub to a different exit (upstream) device.
// this upstream device can either be a physical free device, or zos bridge.
// the method is idempotent.
func SetPublicExitLink(link netlink.Link) error {
	// we can only attach to either physical nic, or zos bridge
	log.Debug().
		Str("type", link.Type()).
		Str("name", link.Attrs().Name).
		Int("master", link.Attrs().MasterIndex).
		Msg("trying to set public exit interface")

	if link.Type() != "device" && link.Attrs().Name != types.DefaultBridge {
		return fmt.Errorf("invalid exit bridge must be a physical nic or the default bridge")
	}

	br, err := bridge.Get(PublicBridge)
	if err != nil {
		return err
	}

	if link.Type() == "device" {
		// already attached
		if link.Attrs().MasterIndex == br.Index {
			return nil
		} else if link.Attrs().MasterIndex != 0 {
			return fmt.Errorf("device is '%s' already used", link.Attrs().Name)
		}
	}

	current, err := GetCurrentPublicExitLink()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if current != nil {
		log.Debug().Str("type", current.Type()).Str("name", current.Attrs().Name).Msg("current attached exit is")

		// disconnect br pub based on the type of the current uplink

		if veth, _ := bootstrap.VEthFilter(current); veth {
			// br pub is already connected to zos
			if link.Attrs().Name == "zos" {
				return nil
			}

			// otherwise we remove this link
			if err := netlink.LinkDel(current); err != nil {
				return errors.Wrap(err, "failed to unhook public bridge from zos bridge")
			}
		} else if err := netlink.LinkSetMasterByIndex(current, 0); err != nil {
			// otherwise we try to remove the nic from br-pub
			return errors.Wrap(err, "failed to unhook public bridge from physical nic")
		}
	}

	return attachPublicToExit(br, link)
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

// EnsurePublicSetup create the public setup, it's okay to have inf == nil.
// this method need to be called at least once in the life of the node. to make bridges are created
// and wired correctly, and initialize public name space if `inf` is found.
// Public bridge wiring (to exit nic) is remembered from last boot. If no exit nic is set
// the node tries to detect the exit br-pub nic based on the following criteria
// - physical nic
// - wired and has a signal
// - can get public slaac IPv6
//
// if no nic is found zos is selected.
// changes to the br-pub exit nic can then be done later with SetPublicExitLink
func EnsurePublicSetup(nodeID pkg.Identifier, inf *pkg.PublicConfig) (*netlink.Bridge, error) {
	log.Debug().Msg("ensure public setup")
	br, err := ensurePublicBridge()
	if err != nil {
		return nil, err
	}

	_, err = GetCurrentPublicExitLink()
	if os.IsNotExist(err) {
		// bridge is not initialized, wire it.
		log.Debug().Msg("no public bridge uplink found, setting up...")
		if err := setupPublicBridge(br); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get current public bridge uplink")
	}

	if inf == nil || inf.IsEmpty() {
		// we need to check if there is already a public config
		// if yes! we need to make sure to delete it and also restart
		// the node because that's the only way to properly unset public config
		_ = DeletePublicConfig()
		if HasPublicSetup() {
			// full node reboot is needed unfortunately
			// to many things depends on the public namespace
			// and the best way to make sure all is in the right state is to
			// reboot the node
			// deleting the namespace alone won't be sufficient because other services
			// live in that namespace (yggdrasile, gateway, and probably other services)
			// also listning wireguards for user networks are inside this namespace.
			// so restarting is the cleanest way to get things in order.
			zi := zinit.Default()
			return nil, zi.Reboot()
		}
	} else {
		if err := setupPublicNS(nodeID, inf); err != nil {
			return nil, errors.Wrap(err, "failed to ensure public namespace setup")
		}
	}

	return br, netlink.LinkSetUp(br)
}

func detectExitNic() (string, error) {
	log.Debug().Msg("find possible ipv6 exit interface")
	// otherwise we try to find the right one
	links, err := bootstrap.AnalyzeLinks(
		bootstrap.RequiresIPv6,
		bootstrap.PhysicalFilter,
		bootstrap.NotAttachedFilter,
		bootstrap.PluggedFilter,
	)

	log.Debug().Int("found", len(links)).Msg("found possible links")
	if err != nil {
		return "", errors.Wrap(err, "failed to analyze links")
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

func ensurePublicNamespace() (ns.NetNS, error) {
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

func ensurePublicResolve() error {
	path := filepath.Join("/etc", "netns", PublicNamespace)
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, "failed to create public netns directory")
	}
	path = filepath.Join(path, "resolv.conf")
	return os.WriteFile(path, []byte(defaultPublicResolveConf), 0644)
}

// setupPublicNS creates a public namespace in a node
func setupPublicNS(nodeID pkg.Identifier, iface *pkg.PublicConfig) error {
	pubNS, err := ensurePublicNamespace()
	if err != nil {
		return err
	}

	defer pubNS.Close()

	// todo: this need to come later from the node config on the grid
	if err := ensurePublicResolve(); err != nil {
		return errors.Wrap(err, "failed to configure public namespace resolv.conf")
	}

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
