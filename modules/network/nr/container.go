package nr

import (
	"fmt"
	"net"

	"golang.org/x/crypto/blake2b"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/vishvananda/netlink"
)

// Join make a network namespace of a container join a network resource network
func (nr *NetResource) Join(containerID string) (join modules.Member, err error) {
	br, err := bridge.Get(nr.nibble.BridgeName())
	if err != nil {
		return join, err
	}

	join.Namespace = containerID
	netspace, err := namespace.Create(containerID)
	if err != nil {
		return join, err
	}

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

		log.Info().
			Str("namespace", join.Namespace).
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

		addr := containerIP(nr.resource.Prefix, containerID)
		if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: addr}); err != nil {
			return err
		}

		join.IP = addr.IP
		return netlink.RouteAdd(&netlink.Route{Gw: nr.resource.Prefix.IP})
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

// this IP is generated for IPv6 only so for IPv4 we still need to set an IP from IPAM
func containerIP(prefix *net.IPNet, containerID string) *net.IPNet {
	h := blake2b.Sum512([]byte(containerID))

	b := make([]byte, net.IPv6len)
	copy(b, prefix.IP)
	copy(b[9:], h[:])

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(64, 128),
	}
}
