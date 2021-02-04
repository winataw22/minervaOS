package nr

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

// ContainerConfig is an object used to pass the required network configuration
// for a container
type ContainerConfig struct {
	ContainerID string
	IPs         []net.IP
	PublicIP6   bool //true if the container must have a public ipv6
	IPv4Only    bool // describe the state of the node, true mean it runs in ipv4 only mode
}

// Join make a network namespace of a container join a network resource network
func (nr *NetResource) Join(cfg ContainerConfig) (join pkg.Member, err error) {
	name, err := nr.BridgeName()
	if err != nil {
		return join, err
	}

	br, err := bridge.Get(name)
	if err != nil {
		return join, err
	}

	join.Namespace = cfg.ContainerID
	netspace, err := namespace.Create(cfg.ContainerID)
	if err != nil {
		return join, err
	}

	slog := log.With().
		Str("namespace", cfg.ContainerID).
		Str("container", cfg.ContainerID).
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

		for _, addr := range cfg.IPs {
			slog.Info().
				Str("ip", addr.String()).
				Msgf("set ip to container")

			if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: &net.IPNet{
				IP:   addr,
				Mask: net.CIDRMask(24, 32),
			}}); err != nil && !os.IsExist(err) {
				return err
			}
			join.IPv4 = addr
		}

		// if the node is IPv6 enabled and the user do not requires a public IPv6
		// then we create derive one to allow IPv6 traffic to go out
		// if the user ask for a public IPv6, then the all config comes from SLAAC so we don't have to do anything ourself
		if !cfg.IPv4Only && !cfg.PublicIP6 {
			ipv6 := Convert4to6(nr.ID(), cfg.IPs[0])
			slog.Info().
				Str("ip", ipv6.String()).
				Msgf("set ip to container")

			if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: &net.IPNet{
				IP:   ipv6,
				Mask: net.CIDRMask(64, 128),
			}}); err != nil && !os.IsExist(err) {
				return err
			}
			join.IPv6 = ipv6
		}

		ipnet := nr.resource.Subnet
		//sanity check this should be already handle by validate.
		//but in case something went wrong.
		if len(ipnet.IP) == 0 {
			return fmt.Errorf("invalid network resource (%s): empty subnet", nr.id)
		}

		ipnet.IP[len(ipnet.IP)-1] = 0x01

		routes := []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("0.0.0.0"),
					Mask: net.CIDRMask(0, 32),
				},
				Gw:        ipnet.IP,
				LinkIndex: eth0.Attrs().Index,
			},
		}

		// same logic as before, we set ipv6 routes only if this is required
		if !cfg.IPv4Only && !cfg.PublicIP6 {
			routes = append(routes,
				&netlink.Route{
					Dst: &net.IPNet{
						IP:   net.ParseIP("::"),
						Mask: net.CIDRMask(0, 128),
					},
					Gw:        net.ParseIP("fe80::1"),
					LinkIndex: eth0.Attrs().Index,
				})
		}

		for _, r := range routes {
			slog.Info().
				Str("route", r.String()).
				Msgf("set route to container")
			err = netlink.RouteAdd(r)
			if err != nil && !os.IsExist(err) {
				return errors.Wrapf(err, "failed to set route %s on eth0", r.String())
			}
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

	if err := options.Set(hostVeth.Attrs().Name, options.IPv6Disable(true)); err != nil {
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

	namespc, err := namespace.GetByName(containerID)
	if _, ok := err.(ns.NSPathNotExistErr); ok {
		return nil
	} else if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer namespc.Close()

	err = namespace.Delete(namespc)
	if err != nil {
		return err
	}
	return nil
}
