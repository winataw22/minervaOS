package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

const (
	ipv4InterfaceArpProxySysctlTemplate = "net.ipv4.conf.%s.proxy_arp"
)

// PublicNamespace is the name of the public namespace of a node
// the public namespace is currently uniq for a node so we hardcode its name
const PublicNamespace = "public"

// type ExitNode struct {
// 	PublicPrefix *net.IPNet
// }

// IfaceType define the different public interface
// supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// ExitIface is the configuration of the interface
// that is connected to the public internet
type ExitIface struct {
	Master string
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType
	Vlan int16
	// Macvlan net.HardwareAddr

	IPv4 *net.IPNet
	IPv6 *net.IPNet

	GW4 net.IP
	GW6 net.IP
}

func createPublicNS(iface *ExitIface) error {
	// create net ns
	// configure the public interface inside the namespace

	log.Info().Str("namespace", PublicNamespace).Msg("Create network namespace")
	pubNS, err := namespace.Create(PublicNamespace)
	if err != nil {
		return err
	}
	defer pubNS.Close()

	var pubIface *current.Interface

	switch iface.Type {
	case MacVlanIface:
		pubIface, err = createMacVlan(iface.Master, pubNS)
		if err != nil {
			log.Error().Err(err).Msg("failed to create public mac vlan interface")
			return err
		}
	default:
		return fmt.Errorf("unsupported iface type %s", iface.Type)
	}

	if err := configurePubIface(pubIface.Name, iface.IPv6, iface.GW6, pubNS); err != nil {
		log.Error().Err(err).Msg("failed to configure public interface")
		return err
	}

	return nil
}

func createMacVlan(master string, netns ns.NetNS) (*current.Interface, error) {
	macvlan := &current.Interface{}

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

		err := ip.RenameLink(tmpName, "public")
		if err != nil {
			_ = netlink.LinkDel(mv)
			return fmt.Errorf("failed to rename macvlan to %q: %v", "public", err)
		}
		macvlan.Name = "public"

		// Re-fetch macvlan to get all properties/attributes
		contMacvlan, err := netlink.LinkByName("public")
		if err != nil {
			return fmt.Errorf("failed to refetch macvlan %q: %v", "public", err)
		}
		macvlan.Mac = contMacvlan.Attrs().HardwareAddr.String()
		macvlan.Sandbox = netns.Path()

		return nil
	})
	if err != nil {
		return nil, err
	}

	return macvlan, nil
}

func configurePubIface(name string, ip *net.IPNet, gw net.IP, netns ns.NetNS) error {
	err := netns.Do(func(_ ns.NetNS) error {
		pubLink, err := netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("failed to find interface name %q: %v", name, err)
		}

		addr := &netlink.Addr{
			IPNet: ip,
		}

		if err := netlink.AddrAdd(pubLink, addr); err != nil {
			return err
		}

		if err := netlink.LinkSetUp(pubLink); err != nil {
			return fmt.Errorf("failed to set %q UP: %v", name, err)
		}

		if err := netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        gw,
			LinkIndex: pubLink.Attrs().Index,
		}); err != nil {
			return err
		}

		return nil
	})
	return err
}
