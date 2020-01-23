package ndmz

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg/network/nr"

	"github.com/threefoldtech/zos/pkg/network/dhcp"

	"github.com/threefoldtech/zos/pkg/network/macvlan"

	"github.com/threefoldtech/zos/pkg/network/bridge"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
)

const (
	//BridgeNDMZ is the name of the ipv4 routing bridge in the ndmz namespace
	BridgeNDMZ = "br-ndmz"
	//NetNSNDMZ name of the dmz namespace
	NetNSNDMZ = "ndmz"

	ndmzNsMACDerivationSuffix = "-ndmz"

	// PublicIfaceName interface name of dmz
	PublicIfaceName = "public"
)

//Create create the NDMZ network namespace and configure its default routes and addresses
func Create(nodeID pkg.Identifier) error {

	os.RemoveAll("/var/cache/modules/networkd/lease/dmz/")

	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err != nil {
		netNS, err = namespace.Create(NetNSNDMZ)
		if err != nil {
			return err
		}
	}

	defer netNS.Close()

	if err := createRoutingBridge(netNS); err != nil {
		return errors.Wrapf(err, "ndmz: createRoutingBride error")
	}

	if err := createPublicIface(netNS); err != nil {
		return errors.Wrapf(err, "ndmz: createPublicIface error")
	}

	// set mac address to something static to make sure we receive the same IP from a DHCP server
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + ndmzNsMACDerivationSuffix))
	log.Debug().
		Str("mac", mac.String()).
		Msg("set mac on public iface")

	if err = ifaceutil.SetMAC(types.PublicIface, mac, netNS); err != nil {
		return err
	}

	if err = applyFirewall(); err != nil {
		return err
	}

	return netNS.Do(func(_ ns.NetNS) error {
		// first, disable forwarding, so we can get an IPv6 deft route on public from an RA
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "0"); err != nil {
			return errors.Wrapf(err, "ndmz: failed to disable ipv6 forwarding in ndmz namespace")
		}
		// run DHCP to interface public in ndmz
		received, err := dhcp.Probe(types.PublicIface, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		if !received {
			return errors.Errorf("public interface in ndmz did not received an IP. make sure dhcp is working")
		}

		var routes []netlink.Route
		getRoutes := func() (err error) {
			log.Info().Msg("wait for slaac to give ipv6")
			// check if in the mean time SLAAC gave us an IPv6 deft gw, save it, and reapply after enabling forwarding
			checkipv6 := net.ParseIP("2606:4700:4700::1111")
			routes, err = netlink.RouteGet(checkipv6)
			if err != nil {
				return errors.Wrapf(err, "ndmz: failed to get the IPv6 routes in ndmz")
			}
			return nil
		}

		bo := backoff.NewExponentialBackOff()
		bo.MaxElapsedTime = 15 * time.Second
		if err := backoff.Retry(getRoutes, bo); err != nil {
			return err
		}

		if len(routes) == 1 {
			if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
				return errors.Wrapf(err, "ndmz: failed to enable ipv6 forwarding in ndmz namespace")
			}
			pubiface, err := netlink.LinkByName(types.PublicIface)
			if err != nil {
				return errors.Wrapf(err, "ndmz:couldn't find public iface")
			}
			deftgw := &netlink.Route{
				Dst: &net.IPNet{
					IP:   net.ParseIP("::"),
					Mask: net.CIDRMask(0, 128),
				},
				Gw:        routes[0].Gw,
				LinkIndex: pubiface.Attrs().Index,
			}
			if err = netlink.RouteAdd(deftgw); err != nil {
				return errors.Wrapf(err, "could not reapply the default route")
			}
		}

		return ifaceutil.SetLoUp()
	})
}

// Delete deletes the NDMZ network namespace
func Delete() error {
	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err == nil {
		if err := namespace.Delete(netNS); err != nil {
			return errors.Wrap(err, "failed to delete ndmz network namespace")
		}
	}

	return nil
}

func createPublicIface(netNS ns.NetNS) error {
	var pubIface string
	if !ifaceutil.Exists(types.PublicIface, netNS) {

		// find which interface to use as master for the macvlan
		if namespace.Exists(types.PublicNamespace) {
			pubNS, err := namespace.GetByName(types.PublicNamespace)
			if err != nil {
				return err
			}
			defer pubNS.Close()

			var ifaceIndex int
			// get the name of the public interface in the public namespace
			if err := pubNS.Do(func(_ ns.NetNS) error {
				// get the name of the interface connected to the public segment
				public, err := netlink.LinkByName(types.PublicIface)
				if err != nil {
					return errors.Wrap(err, "failed to get public link")
				}

				ifaceIndex = public.Attrs().ParentIndex
				return nil
			}); err != nil {
				return err
			}

			master, err := netlink.LinkByIndex(ifaceIndex)
			if err != nil {
				return errors.Wrapf(err, "failed to get link by index %d", ifaceIndex)
			}
			pubIface = master.Attrs().Name
		} else {
			found, err := ifaceutil.HostIPV6Iface()
			if err != nil {
				return errors.Wrap(err, "failed to find a valid network interface to use as parent for ndmz public interface")
			}
			pubIface = found
		}

		_, err := macvlan.Create(types.PublicIface, pubIface, netNS)
		return err
	}

	return nil
}

func createRoutingBridge(netNS ns.NetNS) error {
	if !bridge.Exists(BridgeNDMZ) {
		if _, err := bridge.New(BridgeNDMZ); err != nil {
			return errors.Wrapf(err, "couldn't create bridge %s", BridgeNDMZ)
		}
	}

	const tonrsIface = "tonrs"

	if !ifaceutil.Exists(tonrsIface, netNS) {
		if _, err := macvlan.Create(tonrsIface, BridgeNDMZ, netNS); err != nil {
			return errors.Wrapf(err, "ndmz: couldn't create %s", tonrsIface)
		}
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", BridgeNDMZ), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", BridgeNDMZ)
	}

	return netNS.Do(func(_ ns.NetNS) error {

		link, err := netlink.LinkByName(tonrsIface)
		if err != nil {
			return err
		}
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", tonrsIface), "0"); err != nil {
			return errors.Wrapf(err, "failed to enable ip6 on interface %s", tonrsIface)
		}

		addrs := []*netlink.Addr{
			&netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("100.127.0.1"),
					Mask: net.CIDRMask(16, 32),
				},
			},
			&netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("fe80::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
			&netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("fd00::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
		}

		for _, addr := range addrs {
			err = netlink.AddrAdd(link, addr)
			if err != nil && !os.IsExist(err) {
				return err
			}
		}

		return netlink.LinkSetUp(link)
	})
}

func applyFirewall() error {
	buf := bytes.Buffer{}

	if err := fwTmpl.Execute(&buf, nil); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, NetNSNDMZ); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

// AttachNR links a network resource to the NDMZ
func AttachNR(networkID string, nr *nr.NetResource) error {
	nrNSName, err := nr.Namespace()
	if err != nil {
		return err
	}

	nrNS, err := namespace.GetByName(nrNSName)
	if err != nil {
		return err
	}

	if !ifaceutil.Exists(PublicIfaceName, nrNS) {
		if _, err = macvlan.Create(PublicIfaceName, BridgeNDMZ, nrNS); err != nil {
			return err
		}
	}

	return nrNS.Do(func(_ ns.NetNS) error {
		addr, err := allocateIPv4(networkID)
		if err != nil {
			return errors.Wrap(err, "ip allocation for network resource")
		}

		pubIface, err := netlink.LinkByName(PublicIfaceName)
		if err != nil {
			return err
		}

		if err := netlink.AddrAdd(pubIface, &netlink.Addr{IPNet: addr}); err != nil && !os.IsExist(err) {
			return err
		}

		ipv6 := convertIpv4ToIpv6(addr.IP)
		log.Debug().Msgf("ndmz: setting public NR ip to: %s from %s", ipv6.String(), addr.IP.String())

		if err := netlink.AddrAdd(pubIface, &netlink.Addr{IPNet: &net.IPNet{
			IP:   ipv6,
			Mask: net.CIDRMask(64, 128),
		}}); err != nil && !os.IsExist(err) {
			return err
		}

		if err = netlink.LinkSetUp(pubIface); err != nil {
			return err
		}

		err = netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        net.ParseIP("100.127.0.1"),
			LinkIndex: pubIface.Attrs().Index,
		})
		if err != nil && !os.IsExist(err) {
			return err
		}

		err = netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        net.ParseIP("fe80::1"),
			LinkIndex: pubIface.Attrs().Index,
		})
		if err != nil && !os.IsExist(err) {
			return err
		}

		return nil
	})
}

func convertIpv4ToIpv6(ip net.IP) net.IP {
	var ipv6 string
	if len(ip) == net.IPv4len {
		ipv6 = fmt.Sprintf("fd00::%02x%02x", ip[2], ip[3])
	} else {
		ipv6 = fmt.Sprintf("fd00::%02x%02x", ip[14], ip[15])
	}
	fmt.Println(ipv6)
	return net.ParseIP(ipv6)
}
