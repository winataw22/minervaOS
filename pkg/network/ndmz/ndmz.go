package ndmz

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/termie/go-shutil"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
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

	ndmzNsMACDerivationSuffix6 = "-ndmz6"
	ndmzNsMACDerivationSuffix4 = "-ndmz4"

	// DMZPub4 ipv4 public interface
	DMZPub4 = "npub4"
	// DMZPub6 ipv6 public interface
	DMZPub6 = "npub6"

	//nrPubIface is the name of the public interface in a network resource
	nrPubIface = "public"
)

var ipamPath = "/var/cache/modules/networkd/lease"

//Create create the NDMZ network namespace and configure its default routes and addresses
func Create(nodeID pkg.Identifier) error {
	path := filepath.Join(ipamPath, "ndmz")

	if app.IsFirstBoot("networkd-dmz") {
		log.Info().Msg("first boot, empty reservation cache")

		if err := os.RemoveAll(path); err != nil {
			return err
		}

		// TODO @zaibon: remove once all the network has applies this fix

		// This check ensure we do not delete current lease from the node
		// running a previous version of this code.
		// if the old leases directory exists, we copy the content
		// into the new location and delete the old directory
		var err error
		oldFolder := filepath.Join(ipamPath, "dmz")
		if _, err = os.Stat(oldFolder); err == nil {
			if err := shutil.CopyTree(oldFolder, path, nil); err != nil {
				return err
			}
			err = os.RemoveAll(oldFolder)
		} else {
			err = os.MkdirAll(path, 0770)
		}
		if err != nil {
			return err
		}

		if err := app.MarkBooted("networkd-dmz"); err != nil {
			return errors.Wrap(err, "fail to mark provisiond as booted")
		}

	} else {
		log.Info().Msg("restart detected, keep IPAM lease cache intact")
	}

	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err != nil {
		netNS, err = namespace.Create(NetNSNDMZ)
		if err != nil {
			return err
		}
	}

	defer netNS.Close()

	if err := createRoutingBridge(BridgeNDMZ, netNS); err != nil {
		return errors.Wrapf(err, "ndmz: createRoutingBride error")
	}

	if err := createPubIface6(DMZPub6, netNS, nodeID); err != nil {
		return errors.Wrapf(err, "ndmz: could not node create pub iface 6")
	}

	if err := createPubIface4(DMZPub4, netNS, nodeID); err != nil {
		return errors.Wrapf(err, "ndmz: could not create pub iface 4")
	}

	if err = applyFirewall(); err != nil {
		return err
	}

	return netNS.Do(func(_ ns.NetNS) error {
		if err := ifaceutil.SetLoUp(); err != nil {
			return errors.Wrapf(err, "ndmz: couldn't bring lo up in ndmz namespace")
		}
		// first, disable forwarding, so we can get an IPv6 deft route on public from an RA
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "0"); err != nil {
			return errors.Wrapf(err, "ndmz: failed to disable ipv6 forwarding in ndmz namespace")
		}
		// also, set kernel parameter that public always accepts an ra even when forwarding
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.accept_ra", DMZPub6), "2"); err != nil {
			return errors.Wrapf(err, "ndmz: failed to accept_ra=2 in ndmz namespace")
		}
		// the more, also accept defaultrouter (if isp doesn't have fe80::1 on his deft gw)
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.accept_ra_defrtr", DMZPub6), "1"); err != nil {
			return errors.Wrapf(err, "ndmz: failed to enable enable_defrtr=1 in ndmz namespace")
		}
		// ipv4InterfaceArpProxySysctlTemple sets proxy_arp by default, not sure if that's a good idea
		// but we disable only here because the rest works.
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv4.conf.%s.proxy_arp", DMZPub6), "0"); err != nil {
			return errors.Wrapf(err, "ndmz: couldn't disable proxy-arp on %s in ndmz namespace", DMZPub6)
		}
		// run DHCP to interface public in ndmz
		probe := dhcp.NewProbe()

		if err := probe.Start(DMZPub4); err != nil {
			return err
		}
		defer probe.Stop()

		link, err := netlink.LinkByName(DMZPub4)
		if err != nil {
			return err
		}

		cTimeout := time.After(time.Second * 30)
	Loop:
		for {
			select {
			case <-cTimeout:
				return errors.Errorf("public interface in ndmz did not received an IP. make sure DHCP is working")
			default:
				hasGW, _, err := ifaceutil.HasDefaultGW(link, netlink.FAMILY_V4)
				if err != nil {
					return err
				}
				if !hasGW {
					time.Sleep(time.Second)
					continue
				}
				break Loop
			}
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
		bo.MaxElapsedTime = 122 * time.Second // default RA from router is every 60 secs
		if err := backoff.Retry(getRoutes, bo); err != nil {
			return err
		}

		if len(routes) == 1 {
			if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
				return errors.Wrapf(err, "ndmz: failed to enable ipv6 forwarding in ndmz namespace")
			}
		}
		return nil

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

func createPubIface6(name string, netNS ns.NetNS, nodeID pkg.Identifier) error {
	var pubIface string
	if !ifaceutil.Exists(name, netNS) {

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

		if _, err := macvlan.Create(name, pubIface, netNS); err != nil {
			return err
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		// set mac address to something static to make sure we receive the same IP from a DHCP server
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + ndmzNsMACDerivationSuffix6))
		log.Debug().
			Str("mac", mac.String()).
			Str("interface", name).
			Msg("set mac on ipv6 ndmz public iface")

		if err := ifaceutil.SetMAC(name, mac, nil); err != nil {
			return err
		}

		link, err := netlink.LinkByName(name)
		if err != nil {
			return err
		}
		return netlink.LinkSetUp(link)
	})
}

func createPubIface4(name string, netNS ns.NetNS, nodeID pkg.Identifier) error {
	if !ifaceutil.Exists(name, netNS) {
		if _, err := macvlan.Create(name, types.DefaultBridge, netNS); err != nil {
			return err
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on %s", name)
		}
		// set mac address to something static to make sure we receive the same IP from a DHCP server
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + ndmzNsMACDerivationSuffix4))
		log.Debug().
			Str("mac", mac.String()).
			Str("interface", name).
			Msg("set mac on ipv4 ndmz public iface")

		return ifaceutil.SetMAC(name, mac, nil)
	})
}

func createRoutingBridge(name string, netNS ns.NetNS) error {
	if !bridge.Exists(name) {
		if _, err := bridge.New(name); err != nil {
			return errors.Wrapf(err, "couldn't create bridge %s", name)
		}
	}

	const tonrsIface = "tonrs"

	if !ifaceutil.Exists(tonrsIface, netNS) {
		if _, err := macvlan.Create(tonrsIface, name, netNS); err != nil {
			return errors.Wrapf(err, "ndmz: couldn't create %s", tonrsIface)
		}
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
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

	if !ifaceutil.Exists(nrPubIface, nrNS) {
		if _, err = macvlan.Create(nrPubIface, BridgeNDMZ, nrNS); err != nil {
			return err
		}
	}

	return nrNS.Do(func(_ ns.NetNS) error {
		addr, err := allocateIPv4(networkID)
		if err != nil {
			return errors.Wrap(err, "ip allocation for network resource")
		}

		pubIface, err := netlink.LinkByName(nrPubIface)
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
