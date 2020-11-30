package network

import (
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	publicNsMACDerivationSuffix = "-public"
)

func ensureNamespace() (ns.NetNS, error) {
	if !namespace.Exists(types.PublicNamespace) {
		log.Info().Str("namespace", types.PublicNamespace).Msg("Create network namespace")
		return namespace.Create(types.PublicNamespace)
	}

	return namespace.GetByName(types.PublicNamespace)
}

func ensurePublicMacvlan(iface *types.PubIface, pubNS ns.NetNS) (*netlink.Macvlan, error) {
	var (
		pubIface *netlink.Macvlan
		err      error
	)

	if !ifaceutil.Exists(types.PublicIface, pubNS) {

		switch iface.Type {
		case types.MacVlanIface:
			pubIface, err = macvlan.Create(types.PublicIface, iface.Master, pubNS)
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

func publicConfig(iface *types.PubIface, nodeID pkg.Identifier) (ips []*net.IPNet, routes []*netlink.Route, err error) {
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

// CreatePublicNS creates a public namespace in a node
func CreatePublicNS(dmz ndmz.DMZ, iface *types.PubIface, nodeID pkg.Identifier) error {

	pubNS, err := ensureNamespace()
	if err != nil {
		return err
	}
	defer pubNS.Close()

	// Override master from config to ndmz master.
	// This is fine anyhow since in old nodes the master will be the physical iface
	// and this will be a no-op
	iface.Master = dmz.IP6PublicIface()

	pubIface, err := ensurePublicMacvlan(iface, pubNS)
	if err != nil {
		return err
	}

	log.Info().
		Str("pub iface", fmt.Sprintf("%+v", pubIface)).
		Msg("configure public interface inside public namespace")

	ips, routes, err := publicConfig(iface, nodeID)
	if err != nil {
		return errors.Wrap(err, "failed configure public network interface")
	}

	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + publicNsMACDerivationSuffix))
	if err := macvlan.Install(pubIface, mac, ips, routes, pubNS); err != nil {
		return err
	}

	err = pubNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra", "2"); err != nil {
			return errors.Wrapf(err, "failed to accept_ra=2 in public namespace")
		}
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra_defrtr", "1"); err != nil {
			return errors.Wrapf(err, "failed to enable enable_defrtr=1 in public namespace")
		}
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "error while configuring IPv6 public namespace")
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
