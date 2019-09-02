package network

import (
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/vishvananda/netlink"
)

// CreatePublicNS creates a public namespace in a node
func CreatePublicNS(iface *types.PubIface) error {
	var (
		pubNS    ns.NetNS
		pubIface *netlink.Macvlan
		err      error
	)

	if !namespace.Exists(types.PublicNamespace) {

		log.Info().Str("namespace", types.PublicNamespace).Msg("Create network namespace")

		pubNS, err = namespace.Create(types.PublicNamespace)
		if err != nil {
			return err
		}
		defer pubNS.Close()

		switch iface.Type {
		case types.MacVlanIface:
			pubIface, err = macvlan.Create(types.PublicIface, iface.Master, pubNS)
			if err != nil {
				return errors.Wrap(err, "failed to create public mac vlan interface")
			}
		default:
			return fmt.Errorf("unsupported public interface type %s", iface.Type)
		}
	} else {

		pubNS, err = namespace.GetByName(types.PublicNamespace)
		if err != nil {
			return err
		}
		defer pubNS.Close()

		if err := pubNS.Do(func(_ ns.NetNS) error {
			pubIface, err = macvlan.GetByName(types.PublicIface)
			return err
		}); err != nil {
			return err
		}
	}

	var (
		ips    = make([]*net.IPNet, 0)
		routes = make([]*netlink.Route, 0)
	)

	log.Info().
		Str("pub iface", fmt.Sprintf("%+v", pubIface)).
		Msg("configure public interface inside public namespace")

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
		err := fmt.Errorf("missing some information in the exit iface object")
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
