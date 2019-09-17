package nr

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/blake2b"

	"github.com/containernetworking/cni/pkg/types"
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

	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/disk"
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

		addrs := make([]*net.IPNet, 2)

		addrs[0] = nr.containerIPv6(containerID)
		join.IPv6 = addrs[0].IP

		addrs[1], err = nr.allocateIPv4(containerID)
		if err != nil {
			return errors.Wrapf(err, "failed to allocate IPv4 for container %s", containerID)
		}
		join.IPv4 = addrs[1].IP

		for _, addr := range addrs {
			slog.Info().
				IPPrefix("ip", *addr).
				Msgf("set ip to container")
			if err := netlink.AddrAdd(eth0, &netlink.Addr{IPNet: addr}); err != nil && !os.IsExist(err) {
				return err
			}
		}

		routes := make([]*netlink.Route, 2)
		routes[0] = &netlink.Route{Gw: nr.resource.Prefix.IP} // default IPv6 route
		routes[1] = nr.nibble.RouteIPv4DefaultContainer()     // default IPv4 route

		for _, route := range routes {
			slog.Info().
				Str("route", route.String()).
				Msgf("set route to container")
			if err := netlink.RouteAdd(route); err != nil {
				return err
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

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", hostVeth.Attrs().Name), "1"); err != nil {
		return join, errors.Wrapf(err, "failed to disable ip6 on bridge %s", hostVeth.Attrs().Name)
	}

	return join, bridge.AttachNic(hostVeth, br)
}

// containerIPv6 generates an IPv6 for a container linked to a network resource
func (nr *NetResource) containerIPv6(containerID string) *net.IPNet {
	h := blake2b.Sum512([]byte(containerID))

	b := make([]byte, net.IPv6len)
	copy(b, nr.resource.Prefix.IP)
	copy(b[9:], h[:])

	return &net.IPNet{
		IP:   net.IP(b),
		Mask: net.CIDRMask(64, 128),
	}
}

// allocateIPv4 allocates a unique IPv4 for the entity defines by the given id (for example container id, or a vm).
// in the network with netID, and NetResource.
func (nr *NetResource) allocateIPv4(id string) (*net.IPNet, error) {
	// FIXME: path to the cache disk shouldn't be hardcoded here
	store, err := disk.New(string(nr.networkID), "/var/cache/modules/networkd/lease")
	if err != nil {
		return nil, err
	}

	nrIP4 := nr.nibble.NRLocalIP4()
	nrIP4.IP[15] = 0x00

	firstIP := make([]byte, net.IPv6len)
	copy(firstIP, nrIP4.IP)
	firstIP[15] = 0x02

	lastIP := make([]byte, net.IPv6len)
	copy(lastIP, nrIP4.IP)
	lastIP[15] = 0xfe

	log.Debug().Str("ip4 range", nrIP4.String()).Msg("configure ipam range")
	r := allocator.Range{
		RangeStart: net.IP(firstIP),
		RangeEnd:   net.IP(lastIP),
		Subnet:     types.IPNet(*nrIP4),
		Gateway:    nrIP4.IP,
	}

	if err := r.Canonicalize(); err != nil {
		return nil, err
	}

	set := allocator.RangeSet{r}

	// unfortunately, calling the allocator Get() directly will try to allocate
	// a new IP. if the ID/nic already has an ip allocated it will just fail instead of returning
	// the same IP.
	// So we have to check the store ourselves to see if there is already an IP allocated
	// to this container, and if one found, we return it.
	store.Lock()
	ips := store.GetByID(id, "eth0")
	store.Unlock()
	if len(ips) > 0 {
		ip := ips[0]
		rng, err := set.RangeFor(ip)
		if err != nil {
			return nil, err
		}

		return &net.IPNet{IP: ip, Mask: rng.Subnet.Mask}, nil
	}

	aloc := allocator.NewIPAllocator(&set, store, 0)

	ipConfig, err := aloc.Get(id, "eth0", nil)
	if err != nil {
		return nil, err
	}
	return &ipConfig.Address, nil
}
