package macvlan

import (
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/vishvananda/netlink"
)

const ipv4InterfaceArpProxySysctlTemplate = "net.ipv4.conf.%s.proxy_arp"

// Create creates a new macvlan interface in the network namespace
// name is the name of the macvlan interface
// master is the name of the device used as master for the macvlan interface
// netns is network namespace where to create the macvlan
func Create(name string, master string, netns ns.NetNS) (*netlink.Macvlan, error) {

	m, err := netlink.LinkByName(master)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup master %q: %v", master, err)
	}

	// due to kernel bug we have to create with tmpName or it might
	// collide with the name on the host and error out
	tmpName, err := ip.RandomVethName()
	if err != nil {
		return nil, err
	}

	mv := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         1500,
			Name:        tmpName,
			ParentIndex: m.Attrs().Index,
			Namespace:   netlink.NsFd(int(netns.Fd())),
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	if err := netlink.LinkAdd(mv); err != nil {
		return nil, fmt.Errorf("failed to create macvlan: %v", err)
	}

	err = netns.Do(func(_ ns.NetNS) error {
		// TODO: duplicate following lines for ipv6 support, when it will be added in other places
		ipv4SysctlValueName := fmt.Sprintf(ipv4InterfaceArpProxySysctlTemplate, tmpName)
		if _, err := sysctl.Sysctl(ipv4SysctlValueName, "1"); err != nil {
			// remove the newly added link and ignore errors, because we already are in a failed state
			_ = netlink.LinkDel(mv)
			return fmt.Errorf("failed to set proxy_arp on newly added interface %q: %v", tmpName, err)
		}

		err := ip.RenameLink(tmpName, name)
		if err != nil {
			_ = netlink.LinkDel(mv)
			return fmt.Errorf("failed to rename macvlan to %q: %v", name, err)
		}

		// Re-fetch macvlan to get all properties/attributes
		link, err := netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("failed to refetch macvlan %q: %v", name, err)
		}
		var ok bool
		mv, ok = link.(*netlink.Macvlan)
		if !ok {
			return fmt.Errorf("link %s should be of type macvlan", name)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return mv, nil
}

// Install configures a macvlan interfaces created with Create method
func Install(link *netlink.Macvlan, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netns ns.NetNS) error {
	return netns.Do(func(_ ns.NetNS) error {
		if hw != nil && len(hw) != 0 {
			if err := netlink.LinkSetHardwareAddr(link, hw); err != nil {
				return err
			}
		}

		name := link.Attrs().Name

		for _, ip := range ips {
			if err := netlink.AddrAdd(link, &netlink.Addr{
				IPNet: ip,
			}); err != nil && !os.IsExist(err) {
				log.Error().
					Str("addr", ip.String()).
					Str("link", link.Attrs().Name).
					Err(err).Msg("failed to set address on link")
				return err
			}
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to set %q UP: %v", name, err)
		}

		for _, route := range routes {
			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
				log.Error().
					Str("route", route.String()).
					Str("link", link.Attrs().Name).
					Err(err).Msg("failed to set route on link")
				return err
			}
		}

		return nil
	})
}

// GetByName return a macvlan object by its name
func GetByName(name string) (*netlink.Macvlan, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	macvlan, ok := link.(*netlink.Macvlan)
	if !ok {
		return nil, fmt.Errorf("link %s is not a macvlan", name)
	}
	return macvlan, nil
}
