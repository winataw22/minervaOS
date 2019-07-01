package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
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

	Version int
}

// CreatePublicNS creates a public namespace in a node
func CreatePublicNS(iface *ExitIface) error {
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
			log.Error().Err(err).Msg("failed to create public mac vlan interface")
			return err
		}
	default:
		return fmt.Errorf("unsupported iface type %s", iface.Type)
	}

	if iface.IPv6 != nil && iface.GW6 != nil {
		routes := []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("::"),
					Mask: net.CIDRMask(0, 128),
				},
				Gw:        iface.GW6,
				LinkIndex: pubIface.Attrs().Index,
			},
		}
		if err := macvlan.Install(pubIface, iface.IPv6, routes, pubNS); err != nil {
			return err
		}

	} else if iface.IPv4 != nil && iface.GW4 != nil {
		routes := []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("0.0.0.0"),
					Mask: net.CIDRMask(0, 32),
				},
				Gw:        iface.GW4,
				LinkIndex: pubIface.Attrs().Index,
			},
		}
		if err := macvlan.Install(pubIface, iface.IPv4, routes, pubNS); err != nil {
			return err
		}
	} else {
		err = fmt.Errorf("missing some information in the exit iface object")
		log.Error().Err(err).Msg("failed to configure public interface")
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
