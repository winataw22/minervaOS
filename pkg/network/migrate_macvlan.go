package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

const (
	zdbNsPrefix = "zdb-ns-"
	macvlanType = "macvlan"
	mtu         = 1500
)

type Veth struct {
	name      string // iface name inside the namespace
	bridgeIdx int    // bridge index on the host
	addresses []netlink.Addr
	routes    []netlink.Route
}

func (n *networker) MigrateZdbMacvlanToVeth() error {
	netNss, err := namespace.List(zdbNsPrefix)
	if err != nil {
		return fmt.Errorf("failed to list namespaces with prefix %q: %w", zdbNsPrefix, err)
	}
	if len(netNss) != 1 {
		return fmt.Errorf("should find only one namespace with prefix %q, found %v", zdbNsPrefix, len(netNss))
	}

	netNs, err := namespace.GetByName(netNss[0])
	if err != nil {
		return fmt.Errorf("failed to get namespace with name %q: %w", netNss[0], err)
	}
	defer netNs.Close()

	var veths []Veth
	if err := netNs.Do(func(_ ns.NetNS) error {
		veths, err = getOldInterfaces()
		return err
	}); err != nil {
		return fmt.Errorf("failed to get old links from namespace %q: %w", filepath.Base(netNs.Path()), err)
	}

	log.Debug().Msgf("starting to create veths for zdb: %v", veths)

	for _, veth := range veths {
		master, err := netlink.LinkByIndex(veth.bridgeIdx)
		if err != nil {
			return fmt.Errorf("failed to get master by index %v: %w", veth.bridgeIdx, err)
		}

		if err := netNs.Do(func(_ ns.NetNS) error {
			log.Debug().Str("ifname", veth.name).Msg("deleting old link connected with macvlan")
			return deleteOldLink(veth.name)
		}); err != nil {
			return fmt.Errorf("failed to remove old link in namespace: %w", err)
		}

		log.Debug().
			Str("in-namespace", filepath.Base(netNs.Path())).
			Str("from-interface", veth.name).
			Str("to-master-bridge", master.Attrs().Name).
			Msg("creating veth pair")
		if err := ifaceutil.MakeVethPair(veth.name, master.Attrs().Name, mtu, netNs); err != nil {
			return fmt.Errorf("failed to attach with veth from %q to master %q : %w", veth.name, master.Attrs().Name, err)
		}

		if err := netNs.Do(func(_ ns.NetNS) error {
			log.Debug().
				Str("ifname", veth.name).
				Msg("setup address and routes for interface in namespace")
			return setupNamespace(veth.name, veth.addresses, veth.routes)
		}); err != nil {
			return fmt.Errorf("failed to add address in namespace: %w", err)
		}
	}

	return nil
}

// return the macvlan interfaces info to reconstruct as veth
func getOldInterfaces() ([]Veth, error) {
	var veths []Veth

	links, err := netlink.LinkList()
	if err != nil {
		return veths, fmt.Errorf("failed to list links: %w", err)
	}

	macvlanLinks := ifaceutil.LinkFilter(links, []string{macvlanType})
	for _, l := range macvlanLinks {
		addrs, err := netlink.AddrList(l, netlink.FAMILY_ALL)
		if err != nil {
			return veths, fmt.Errorf("failed to list link addresses: %w", err)
		}
		routes, err := netlink.RouteList(l, netlink.FAMILY_ALL)
		if err != nil {
			return veths, fmt.Errorf("failed to list link routes: %w", err)
		}

		veths = append(veths, Veth{
			name:      l.Attrs().Name,
			bridgeIdx: l.Attrs().ParentIndex,
			addresses: addrs,
			routes:    routes,
		})
	}

	return veths, nil
}

func deleteOldLink(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to get link by name %q: %w", name, err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete link by name %q: %w", name, err)
	}

	return nil
}

// add addresses and routes to the named link inside namespace
// also it setup the link up and enable ipv6 forwarding
func setupNamespace(linkName string, addrs []netlink.Addr, routes []netlink.Route) error {
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		err = netlink.AddrAdd(link, &addr)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to add ip address to link: %w", err)
		}
	}

	if err := options.SetIPv6Forwarding(true); err != nil {
		return fmt.Errorf("failed to enable ipv6 forwarding: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set link up %q: %w", link.Attrs().Name, err)
	}

	for _, route := range routes {
		route.LinkIndex = link.Attrs().Index // the index for the new created link
		err := netlink.RouteAdd(&route)
		if err != nil && !os.IsExist(err) { // skip if the routing rule exists
			return fmt.Errorf("failed to add route %v: %w", route, err)
		}
	}

	return nil
}

// for debugging
func (v Veth) String() string {
	var ips, routes []string

	if v.addresses != nil {
		for _, ip := range v.addresses {
			ips = append(ips, ip.String())
		}
	}

	if v.routes != nil {
		for _, route := range v.routes {
			routes = append(routes, route.String())
		}
	}

	return fmt.Sprintf("Veth{Name: %s, BridgeIdx: %d, IPs: [%s], Routes: [%s]}\n\n",
		v.name, v.bridgeIdx, strings.Join(ips, ", "), strings.Join(routes, ", "))
}
