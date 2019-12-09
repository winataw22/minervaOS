package ndmz

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"

	"github.com/threefoldtech/zos/pkg/network/nr"

	"github.com/threefoldtech/zos/pkg/network/dhcp"

	"github.com/threefoldtech/zos/pkg/network/macvlan"

	"github.com/threefoldtech/zos/pkg/network/bridge"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
)

const (
	//BridgeNDMZ is the name of the ipv4 routing bridge in the ndmz namespace
	BridgeNDMZ = "br-ndmz"
	netNSNDMZ  = "ndmz"

	vethGWSide = "ipv4-rt"
	vethBrSide = "to-gw"

	ndmzNsMACDerivationSuffix = "-ndmz"
)

//Create create the NDMZ network namespace and configure its default routes and addresses
func Create(nodeID pkg.Identifier) error {

	os.RemoveAll("/var/cache/modules/networkd/lease/dmz/")

	netNS, err := namespace.GetByName(netNSNDMZ)
	if err != nil {
		netNS, err = namespace.Create(netNSNDMZ)
		if err != nil {
			return err
		}
	}

	defer netNS.Close()

	if err := netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return errors.Wrapf(err, "failed to enable ipv6 forwarding in gateway namespace")
		}
		return nil
	}); err != nil {
		return err
	}

	if err := createRoutingBridge(netNS); err != nil {
		return err
	}

	if namespace.Exists("public") {
		err = createMacVlan(netNS)
	} else {
		err = attachVeth(netNS)
	}
	if err != nil {
		return err
	}

	// set mac address to something static to make sure we receive the same IP from a DHCP server
	pubiface, err := getPublicIface()
	if err != nil {
		return err
	}
	mac, err := ifaceutil.GetMAC(pubiface, nil)
	if err != nil {
		return err
	}

	log.Debug().
		Str("iface", pubiface).
		Str("mac", mac.String()).
		Msg("public iface found")

	mac = ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + ndmzNsMACDerivationSuffix))
	log.Debug().
		Str("mac", mac.String()).
		Msg("set mac on public iface")

	if err = ifaceutil.SetMAC("public", mac, netNS); err != nil {
		return err
	}

	err = netNS.Do(func(_ ns.NetNS) error {
		// run DHCP to interface public in ndmz
		received, err := dhcp.Probe("public")
		if err != nil {
			return err
		}
		if !received {
			return errors.Errorf("public interface in ndmz did not received an IP. make sure dhcp is working")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return applyFirewall()
}

func createMacVlan(netNS ns.NetNS) error {
	if !macvlan.Exists("public", netNS) {
		pubIface, err := getPublicIface()
		if err != nil {
			return err
		}

		_, err = macvlan.Create("public", pubIface, netNS)
		if err != nil {
			return err
		}
	}

	return nil
}

func attachVeth(netNS ns.NetNS) error {

	const (
		vethHost = "to-dmz"
		vethNDMZ = "public"
	)

	if !ifaceutil.Exists(vethHost, nil) || !ifaceutil.Exists(vethNDMZ, netNS) {
		if _, _, err := ip.SetupVethWithName(vethHost, vethNDMZ, 1500, netNS); err != nil {
			return errors.Wrap(err, "failed to create veth pair to connect zos bridge and ndmz bridge")
		}
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", vethHost), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on interface %s", vethHost)
		}
	}
	log.Info().
		Str("ndmz side", vethNDMZ).
		Str("host side", vethHost).
		Msg("veth pair for ndmz to public zos bridge created")

	lVethBr, err := netlink.LinkByName(vethHost)
	if err != nil {
		return err
	}

	br, err := bridge.Get("zos") //TODO: use constant
	if err != nil {
		return err
	}

	if err := bridge.AttachNic(lVethBr, br); err != nil {
		return err
	}

	return nil
}

func createRoutingBridge(netNS ns.NetNS) error {
	if bridge.Exists(BridgeNDMZ) {
		return nil
	}

	br, err := bridge.New(BridgeNDMZ)
	if err != nil {
		return err
	}

	vethNDMZ := "tonrs"
	vethHost := "br-tonrs"

	if _, _, err = ip.SetupVethWithName(vethHost, vethNDMZ, 1500, netNS); err != nil {
		return errors.Wrap(err, "failed to create veth pair for ndmz")
	}
	log.Info().
		Str("ndmz side", vethNDMZ).
		Str("host side", vethHost).
		Msg("veth pair for ndmz bridge created")

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", BridgeNDMZ), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", BridgeNDMZ)
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", vethHost), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on interface %s", vethHost)
	}

	lVethBr, err := netlink.LinkByName(vethHost)
	if err != nil {
		return err
	}

	if err := bridge.AttachNic(lVethBr, br); err != nil {
		return err
	}

	return netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", vethNDMZ), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on veth pair %s", vethNDMZ)
		}

		lVethGW, err := netlink.LinkByName(vethNDMZ)
		if err != nil {
			return err
		}

		return netlink.AddrAdd(lVethGW, &netlink.Addr{
			IPNet: &net.IPNet{
				IP:   net.ParseIP("100.127.0.1"),
				Mask: net.CIDRMask(16, 32),
			},
		})
	})
}

func applyFirewall() error {
	buf := bytes.Buffer{}

	if err := fwTmpl.Execute(&buf, nil); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, netNSNDMZ); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

// AttachNR links a network resource to the DMZ
func AttachNR(networkID string, nr *nr.NetResource) error {
	nrNSName, err := nr.Namespace()
	if err != nil {
		return err
	}

	nrNS, err := namespace.GetByName(nrNSName)
	if err != nil {
		return err
	}

	vethNR := "public"
	vethDMZ := fmt.Sprintf("n-%s", nr.ID())

	if !ifaceutil.Exists(vethDMZ, nil) || !ifaceutil.Exists(vethNR, nrNS) {
		log.Debug().
			Str("nr side", vethNR).
			Str("dmz side", vethDMZ).
			Msg("create veth pair to connect network resource and ndmz")

		_ = ifaceutil.Delete(vethDMZ, nil)
		_ = ifaceutil.Delete(vethNR, nrNS)

		if _, _, err = ip.SetupVethWithName(vethDMZ, vethNR, 1500, nrNS); err != nil {
			return errors.Wrap(err, "failed to create veth pair for to connect network resource and ndmz")
		}
	}

	err = nrNS.Do(func(_ ns.NetNS) error {
		addr, err := allocateIPv4(networkID)
		if err != nil {
			return errors.Wrap(err, "ip allocation for network resource veth error")
		}

		lvethNR, err := netlink.LinkByName((vethNR))
		if err != nil {
			return err
		}

		if err := netlink.AddrAdd(lvethNR, &netlink.Addr{IPNet: addr}); err != nil && !os.IsExist(err) {
			return err
		}

		err = netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        net.ParseIP("100.127.0.1"),
			LinkIndex: lvethNR.Attrs().Index,
		})
		if err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	lVethDMZ, err := netlink.LinkByName(vethDMZ)
	if err != nil {
		return err
	}

	br, err := bridge.Get(BridgeNDMZ)
	if err != nil {
		return err
	}
	if err := bridge.AttachNic(lVethDMZ, br); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "failed to attach veth %s to bridge %s", vethDMZ, BridgeNDMZ)
	}
	return nil
}

// AddNetResource adds the routes of a network resource to the gateway network namespace
// func (gw *NDMZ) AddNetResource(netRes *nr.NetResource) error {
// 	log.Info().Msg("add network resource to gateway namespace")
// 	gwNS, err := namespace.GetByName(types.GatewayNamespace)
// 	if err != nil {
// 		return errors.Wrap(err, "gateway namespace not found")
// 	}
// 	defer gwNS.Close()

// 	netResNS, err := namespace.GetByName(netRes.NamespaceName())
// 	if err != nil {
// 		return errors.Wrapf(err, "namespace %s not found", netRes.NamespaceName())
// 	}
// 	defer netResNS.Close()

// 	if err := gw.configNRIPv6(netRes, gwNS, netResNS); err != nil {
// 		return err
// 	}

// 	if err := gw.configNRIPv4(netRes, gwNS, netResNS); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (gw *NDMZ) configNRIPv6(netRes *nr.NetResource, gwNS, netResNS ns.NetNS) error {

// 	epName := netRes.Nibble().EPPubName()
// 	gwName := netRes.Nibble().GWtoEPName()

// 	if !ifaceutil.Exists(epName, netResNS) || !ifaceutil.Exists(gwName, gwNS) {
// 		log.Warn().Msg("one side of the gateway veth pair does not exists, deleting both")
// 		if err := ifaceutil.Delete(epName, netResNS); err != nil {
// 			return err
// 		}
// 		if err := ifaceutil.Delete(gwName, gwNS); err != nil {
// 			return err
// 		}

// 		log.Info().
// 			Str("gateway side", gwName).
// 			Str("exit point side", epName).
// 			Msg("create a veth pair in the host namespace and send one side into the gateway namespace")
// 		if _, _, err := ip.SetupVethWithName(epName, gwName, 1500, gwNS); err != nil {
// 			return errors.Wrap(err, "failed to create veth pair for gateway namespace")
// 		}
// 		log.Info().
// 			Str("gwVeth", gwName).
// 			Str("epVeth", epName).
// 			Msg("veth pair for gateway and exit point created")

// 		// send the other side inside in the exit point network resource namespace
// 		EPLink, err := netlink.LinkByName(epName)
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to get interface %s", epName)
// 		}
// 		if err = netlink.LinkSetNsFd(EPLink, int(netResNS.Fd())); err != nil {
// 			return fmt.Errorf("failed to move interface %s to network resource netns: %v", epName, err)
// 		}
// 	}

// 	err := gwNS.Do(func(_ ns.NetNS) error {
// 		GWLink, err := netlink.LinkByName(gwName)
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to get interface %s in gateway namespace", gwName)
// 		}

// 		addr := &netlink.Addr{IPNet: netRes.Nibble().GWtoEPLL(), Label: ""}
// 		if err := netlink.AddrAdd(GWLink, addr); err != nil && !os.IsExist(err) {
// 			return err
// 		}

// 		route := netRes.Nibble().GWDefaultRoute()
// 		route.LinkIndex = GWLink.Attrs().Index
// 		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
// 			return err
// 		}

// 		routes, err := netRes.GWTNRoutesIPv6()
// 		if err != nil {
// 			return err
// 		}
// 		for _, route := range routes {
// 			route.LinkIndex = GWLink.Attrs().Index
// 			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
// 				return errors.Wrapf(err, "failed to set route %s on %s", route.String(), GWLink.Attrs().Name)
// 			}
// 		}

// 		return nil
// 	})
// 	if err != nil {
// 		return errors.Wrap(err, "failed to configure veth pair end in gateway namespace")
// 	}

// 	// configure veth pair inside the exit point network resource namespace
// 	err = netResNS.Do(func(_ ns.NetNS) error {
// 		EPLink, err := netlink.LinkByName(epName)
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to get interface %s in exit point namespace", epName)
// 		}

// 		addr := &netlink.Addr{IPNet: netRes.Nibble().EPPubLL(), Label: ""}
// 		if err := netlink.AddrAdd(EPLink, addr); err != nil && !os.IsExist(err) {
// 			return err
// 		}

// 		if err := netlink.LinkSetUp(EPLink); err != nil {
// 			return err
// 		}

// 		route := netRes.Nibble().NRDefaultRoute()
// 		route.LinkIndex = EPLink.Attrs().Index
// 		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
// 			return err
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return errors.Wrap(err, "failed to configure veth pair end in exit point namespace")
// 	}
// 	return nil
// }

// func (gw *NDMZ) configNRIPv4(netRes *nr.NetResource, gwNS, netResNS ns.NetNS) error {
// 	ep4pubName := netRes.Nibble().EP4PubName()
// 	br4pubName := netRes.Nibble().Br4PubName()

// 	if !ifaceutil.Exists(ep4pubName, netResNS) || !ifaceutil.Exists(br4pubName, nil) {
// 		log.Warn().Msg("one side of the gateway veth pair does not exists, deleting both")
// 		if err := ifaceutil.Delete(ep4pubName, netResNS); err != nil {
// 			return err
// 		}
// 		if err := ifaceutil.Delete(br4pubName, nil); err != nil {
// 			return err
// 		}

// 		if _, _, err := ip.SetupVethWithName(br4pubName, ep4pubName, 1500, netResNS); err != nil {
// 			return errors.Wrap(err, "failed to create veth pair for bridge gateway")
// 		}
// 		log.Info().
// 			Str("gatway side", br4pubName).
// 			Str("bridge side", ep4pubName).
// 			Msg("veth pair for ipv4 gateway bridge created")
// 	}

// 	br, err := bridge.Get(BridgeNDMZ)
// 	if err != nil {
// 		return err
// 	}

// 	lVethBr, err := netlink.LinkByName(br4pubName)
// 	if err != nil {
// 		return err
// 	}

// 	if err := bridge.AttachNic(lVethBr, br); err != nil {
// 		return err
// 	}

// 	if err := netResNS.Do(func(_ ns.NetNS) error {
// 		lep4pub, err := netlink.LinkByName(ep4pubName)
// 		if err != nil {
// 			return err
// 		}

// 		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", ep4pubName), "1"); err != nil {
// 			return err
// 		}

// 		addr := &netlink.Addr{IPNet: netRes.Nibble().WGIP4()}
// 		log.Debug().Msgf("set addr %s on %s", addr.IPNet.String(), ep4pubName)
// 		if err := netlink.AddrAdd(lep4pub, addr); err != nil && !os.IsExist(err) {
// 			return err
// 		}

// 		if err := netlink.LinkSetUp(lep4pub); err != nil {
// 			return err
// 		}

// 		route := &netlink.Route{
// 			Dst: &net.IPNet{
// 				IP:   net.ParseIP("0.0.0.0"),
// 				Mask: net.CIDRMask(0, 32),
// 			},
// 			Gw:        net.ParseIP("10.1.0.1"),
// 			LinkIndex: lep4pub.Attrs().Index,
// 		}

// 		log.Debug().Msgf("set route %s on %s %d", route.String(), ep4pubName, lep4pub.Attrs().Index)
// 		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
// 			return err
// 		}
// 		return nil
// 	}); err != nil {
// 		return err
// 	}

// 	return gwNS.Do(func(_ ns.NetNS) error {
// 		l, err := netlink.LinkByName(vethGWSide)
// 		if err != nil {
// 			return err
// 		}

// 		routes, err := netRes.GWTNRoutesIPv4()
// 		if err != nil {
// 			return err
// 		}

// 		for _, route := range routes {
// 			route.LinkIndex = l.Attrs().Index
// 			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
// 				return errors.Wrapf(err, "failed to set route %s on %s", route.String(), l.Attrs().Name)
// 			}
// 		}

// 		// TODO: add IPv4 public IP once farmer can specify ipv4 public address in the farm management
// 		return nil
// 	})
// }
