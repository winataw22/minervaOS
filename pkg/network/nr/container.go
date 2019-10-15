package nr

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/vishvananda/netlink"
)

// Join make a network namespace of a container join a network resource network
func (nr *NetResource) Join(containerID string, addrs []net.IP) (join pkg.Member, err error) {
	name, err := nr.BridgeName()
	if err != nil {
		return join, err
	}

	br, err := bridge.Get(name)
	if err != nil {
		return join, err
	}

	join.Namespace = containerID
	netspace, err := namespace.Create(containerID)
	if err != nil {
		return join, err
	}

	slog := log.With().
		Str("namespace", containerID).
		Str("container", containerID).
		Logger()

	defer func() {
		if err != nil {
			namespace.Delete(netspace)
		}
	}()

	var hostVethName string
	err = netspace.Do(func(host ns.NetNS) error {
		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}

		slog.Info().
			Str("veth", "eth0").
			Msg("Create veth pair in net namespace")
		hostVeth, containerVeth, err := ip.SetupVeth("eth0", 1500, host)
		if err != nil {
			return errors.Wrapf(err, "failed to create veth pair in namespace (%s)", join.Namespace)
		}

		hostVethName = hostVeth.Name

		eth0, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			slog.Info().
				Str("ip", addr.String()).
				Msgf("set ip to container")
			if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: &net.IPNet{
				IP:   addr,
				Mask: net.CIDRMask(24, 32),
			}}); err != nil && !os.IsExist(err) {
				return err
			}
		}

		ipnet := nr.resource.Subnet
		ipnet.IP[len(ipnet.IP)-1] = 0x01

		// default IPv4 route
		route := &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        ipnet.IP,
			LinkIndex: eth0.Attrs().Index,
		}
		slog.Info().
			Str("route", route.String()).
			Msgf("set route to container")
		if err := netlink.RouteAdd(route); err != nil {
			return errors.Wrapf(err, "failed to set route %s on eth0", route.String())
		}

		return nil
	})

	if err != nil {
		return join, err
	}

	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return join, err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", hostVeth.Attrs().Name), "1"); err != nil {
		return join, errors.Wrapf(err, "failed to disable ip6 on bridge %s", hostVeth.Attrs().Name)
	}

	return join, bridge.AttachNic(hostVeth, br)
}

// Leave delete a container network namespace
func (nr *NetResource) Leave(containerID string) error {
	log.Info().
		Str("namespace", containerID).
		Str("container", containerID).
		Msg("delete container network namespace")

	ns, err := namespace.GetByName(containerID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		// nothing to do, early return
		return nil
	}
	defer ns.Close()

	err = namespace.Delete(ns)
	if err != nil {
		return err
	}
	return nil
}
