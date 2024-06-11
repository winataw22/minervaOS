package mycelium

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	// MyceliumNSInf inside the namespace
	MyceliumNSInf  = "nmy6"
	myceliumBridge = types.MyceliumBridge
)

var MyRange = net.IPNet{
	IP:   net.ParseIP("400::"),
	Mask: net.CIDRMask(7, 128),
}

type MyceliumNamespace interface {
	Name() string
	// IsIPv4Only checks if namespace has NO public ipv6 on any of its interfaces
	IsIPv4Only() (bool, error)
	// GetIPs return a list of all IPv6 inside this namespace.
	GetIPs() ([]net.IPNet, error)
	// SetMyIP sets the mycelium ipv6 on the nmy6 iterface.
	SetMyIP(ip net.IPNet, gw net.IP) error
}

// ensureMy Plumbing this ensures that the mycelium plumbing is in place inside this namespace
func ensureMyPlumbing(netNS ns.NetNS) error {
	if !bridge.Exists(myceliumBridge) {
		if _, err := bridge.New(myceliumBridge); err != nil {
			return errors.Wrapf(err, "couldn't create bridge %s", myceliumBridge)
		}
	}

	if err := dumdumHack(); err != nil {
		log.Error().Err(err).Msg("failed to create the dummy hack for mycelium-bridge")
	}

	if !ifaceutil.Exists(MyceliumNSInf, netNS) {
		if _, err := macvlan.Create(MyceliumNSInf, myceliumBridge, netNS); err != nil {
			return errors.Wrapf(err, "couldn't create %s inside", MyceliumNSInf)
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(MyceliumNSInf)
		if err != nil {
			return err
		}

		return netlink.LinkSetUp(link)
	})
}

func dumdumHack() error {
	// dumdum hack. this hack to fix a weird issue with linux kernel
	// 5.10.version 55
	// it seems that the macvlan on a bridge does not bring the bridge
	// up. So we have to plug in a dummy device into myceliumBridge and set
	// the device up to keep the bridge state UP.
	br, err := bridge.Get(myceliumBridge)
	if err != nil {
		return errors.Wrap(err, "failed to get br-my")
	}

	const name = "mydumdum"
	link, err := netlink.LinkByName(name)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		if err := netlink.LinkAdd(&netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				NetNsID: -1,
				TxQLen:  -1,
				Name:    name,
			},
		}); err != nil {
			return err
		}

		link, err = netlink.LinkByName(name)
		if err != nil {
			return errors.Wrap(err, "failed to get mydumdum device")
		}
	} else if err != nil {
		return err
	}

	if err := netlink.LinkSetMaster(link, br); err != nil {
		return err
	}

	return netlink.LinkSetUp(link)
}

func NewMyNamespace(ns string) (MyceliumNamespace, error) {
	myNs, err := namespace.GetByName(ns)
	if err != nil {
		return nil, errors.Wrapf(err, "namespace '%s' not found", ns)
	}
	if err := ensureMyPlumbing(myNs); err != nil {
		return nil, errors.Wrapf(err, "failed to prepare namespace '%s' for mycelium", ns)
	}

	return &myNS{ns}, nil
}

type myNS struct {
	ns string
}

func (d *myNS) Name() string {
	return d.ns
}

func (d *myNS) setGw(gw net.IP) error {
	ipv6routes, err := netlink.RouteList(nil, netlink.FAMILY_V6)
	if err != nil {
		return err
	}

	for _, route := range ipv6routes {
		if route.Dst == nil {
			// default route!
			continue
		}
		if route.Dst.String() == MyRange.String() {
			// we found a match
			if err := netlink.RouteDel(&route); err != nil {
				return err
			}
		}
	}

	// now add route
	return netlink.RouteAdd(&netlink.Route{
		Dst: &MyRange,
		Gw:  gw,
	})
}

func (d *myNS) SetMyIP(subnet net.IPNet, gw net.IP) error {
	netns, err := namespace.GetByName(d.ns)
	if err != nil {
		return err
	}
	defer netns.Close()

	if ip6 := subnet.IP.To16(); ip6 == nil {
		return fmt.Errorf("expecting ipv6 for mycelium interface")
	}

	err = netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(MyceliumNSInf)
		if err != nil {
			return err
		}

		ips, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return err
		}

		for _, ip := range ips {
			if MyRange.Contains(ip.IP) {
				_ = netlink.AddrDel(link, &ip)
			}
		}

		if err := netlink.AddrAdd(link, &netlink.Addr{
			IPNet: &subnet,
		}); err != nil && !os.IsExist(err) {
			return err
		}

		if gw == nil {
			return nil
		}
		// set gw for entire mycelium range

		return d.setGw(gw)
	})
	return err
}

func (n *myNS) GetIPs() ([]net.IPNet, error) {
	netns, err := namespace.GetByName(n.ns)
	if err != nil {
		return nil, err
	}

	defer netns.Close()

	var results []net.IPNet
	err = netns.Do(func(_ ns.NetNS) error {
		links, err := netlink.LinkList()
		if err != nil {
			return errors.Wrap(err, "failed to list interfaces")
		}

		for _, link := range links {
			ips, err := netlink.AddrList(link, netlink.FAMILY_V6)
			if err != nil {
				return err
			}

			for _, ip := range ips {
				results = append(results, *ip.IPNet)
			}
		}

		return nil
	})

	return results, err
}

func (n *myNS) IsIPv4Only() (bool, error) {
	// this is true if DMZPub6 only has local not routable ipv6 addresses
	// DMZPub6
	netNS, err := namespace.GetByName(n.ns)
	if err != nil {
		return false, errors.Wrap(err, "failed to get ndmz namespace")
	}
	defer netNS.Close()

	var ipv4Only bool
	err = netNS.Do(func(_ ns.NetNS) error {
		links, err := netlink.LinkList()
		if err != nil {
			return errors.Wrap(err, "failed to list interfaces")
		}

		for _, link := range links {
			ips, err := netlink.AddrList(link, netlink.FAMILY_V6)
			if err != nil {
				return errors.Wrapf(err, "failed to list '%s' ips", link.Attrs().Name)
			}

			for _, ip := range ips {
				if MyRange.Contains(ip.IP) {
					continue
				}

				if ip.IP.IsGlobalUnicast() && !ip.IP.IsPrivate() {
					// we found a public IPv6 so we are not ipv4 so mycelium can peer
					// with other ipv6 peers
					return nil
				}
			}
		}

		ipv4Only = true
		return nil
	})

	return ipv4Only, err
}
