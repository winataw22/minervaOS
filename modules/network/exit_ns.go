package network

import (
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/ip"

	"github.com/containernetworking/plugins/pkg/ns"

	"github.com/vishvananda/netlink"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

const (
	ipv4InterfaceArpProxySysctlTemplate = "net.ipv4.conf.%s.proxy_arp"
)

// PublicNamespace is the name of the public namespace of a node
// the public namespace is currently uniq for a node so we hardcode its name
const PublicNamespace = "public"

// CreatePublicNS creates a public namespace in a node
func CreatePublicNS(iface *PubIface) error {
	// create net ns
	// configure the public interface inside the namespace

	log.Info().Str("namespace", PublicNamespace).Msg("Create network namespace")
	pubNS, err := namespace.Create(PublicNamespace)
	if err != nil {
		return err
	}
	defer pubNS.Close()

	var pubIface *netlink.Macvlan

	switch iface.Type {
	case MacVlanIface:
		pubIface, err = macvlan.Create("public", iface.Master, pubNS)
		if err != nil {
			return errors.Wrap(err, "failed to create public mac vlan interface")
		}
	default:
		return fmt.Errorf("unsupported public interface type %s", iface.Type)
	}
	var (
		ips    = make([]*net.IPNet, 0)
		routes = make([]*netlink.Route, 0)
	)

	if iface.IPv6 != nil && iface.GW6 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        iface.GW6,
			LinkIndex: pubIface.Attrs().Index,
		})
		ips = append(ips, iface.IPv6)
	}
	if iface.IPv4 != nil && iface.GW4 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        iface.GW4,
			LinkIndex: pubIface.Attrs().Index,
		})
		ips = append(ips, iface.IPv4)
	}

	if len(ips) <= 0 || len(routes) <= 0 {
		err = fmt.Errorf("missing some information in the exit iface object")
		log.Error().Err(err).Msg("failed to configure public interface")
		return err
	}

	if err := macvlan.Install(pubIface, ips, routes, pubNS); err != nil {
		return err
	}

	master, err := netlink.LinkByName(iface.Master)
	if err != nil {
		return err
	}
	if err := netlink.LinkSetUp(master); err != nil {
		return err
	}

	return nil
}

func configNetResAsExitPoint(nr *modules.NetResource, ep *modules.ExitPoint, prefixZero *net.IPNet) error {

	if nr.NodeID.ReachabilityV6 == modules.ReachabilityV6ULA {
		return fmt.Errorf("cannot configure an exit point in a hidden node")
	}

	if prefixZero == nil {
		return fmt.Errorf("prefixZero cannot be nil")
	}

	// TODO
	// if nr.NodeID.ReachabilityV4 == modules.ReachabilityV4Hidden{
	// }

	nibble := ip.NewNibble(nr.Prefix, 0) //FIXME: alloc number not always 0

	pubIface := &current.Interface{}
	pubNS, err := namespace.GetByName("public")

	if err == nil { // there is a public namespace on the node
		var ifaceIndex int

		defer pubNS.Close()
		// get the name of the public interface in the public namespace
		if err := pubNS.Do(func(_ ns.NetNS) error {
			// get the name of the interface connected to the public segment
			public, err := netlink.LinkByName("public")
			if err != nil {
				return errors.Wrap(err, "failed to get public link")
			}

			ifaceIndex = public.Attrs().ParentIndex

			return nil
		}); err != nil {
			return err
		}

		master, err := netlink.LinkByIndex(ifaceIndex)
		if err != nil {
			return errors.Wrapf(err, "failed to get link by index %d", ifaceIndex)
		}
		pubIface.Name = master.Attrs().Name
	} else {
		// since we are a fully public node
		// get the name of the interface that has the default gateway
		links, err := netlink.LinkList()
		if err != nil {
			return errors.Wrap(err, "failed to list interfaces")
		}
		for _, link := range links {
			has, _, err := ifaceutil.HasDefaultGW(link)
			if err != nil {
				return errors.Wrapf(err, "fail to inspect default gateway of iface %s", link.Attrs().Name)
			}
			if !has {
				continue
			}
			pubIface.Name = link.Attrs().Name
			break
		}
	}

	nrNS, err := namespace.GetByName(nibble.NetworkName())
	if err != nil {
		return errors.Wrapf(err, "fail to get network namespace for network %s", nibble.NetworkName())
	}
	defer nrNS.Close()

	pubMacVlan, err := macvlan.Create("public", pubIface.Name, nrNS)
	if err != nil {
		log.Error().Err(err).Msg("failed to create public mac vlan interface")
		return errors.Wrap(err, "failed to create public mac vlan interface")
	}

	var (
		ips    []*net.IPNet
		routes []*netlink.Route
	)

	routes = []*netlink.Route{
		{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        net.ParseIP("fe80::1"),
			LinkIndex: pubMacVlan.Attrs().Index,
		},
	}

	if ep.Ipv6Conf != nil && ep.Ipv6Conf.Addr != nil {
		ips = append(ips, ep.Ipv6Conf.Addr)
	} else {
		ips = append(ips, &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
			Mask: net.CIDRMask(64, 128),
		})
	}

	cidr := fmt.Sprintf("%s%s/64", prefixZero.IP.String(), nibble.Hex())
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.Wrapf(err, "fail to parse CIDR: %v", cidr)
	}
	ipnet.IP = ip
	ips = append(ips, ipnet)

	if err := macvlan.Install(pubMacVlan, ips, routes, nrNS); err != nil {
		return errors.Wrap(err, "fail to install mac vlan")
	}

	return nil
}

func getPublicMasterIface() (netlink.Link, error) {
	// get the name of the interface connected to the public segment
	public, err := netlink.LinkByName("public")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public link")
	}

	index := public.Attrs().MasterIndex
	master, err := netlink.LinkByIndex(index)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get link by index %d", index)
	}
	return master, nil
}
