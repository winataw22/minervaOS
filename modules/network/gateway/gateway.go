package gateway

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/nft"
	"github.com/threefoldtech/zosv2/modules/network/nr"
	"github.com/threefoldtech/zosv2/modules/network/types"
)

const (
	//BridgeGateway is the name of the ipv4 routing bridge in the gateway namespace
	BridgeGateway = "br-gw-4"

	vethGWSide = "ipv4-rt"
	vethBrSide = "to-gw"
)

//Gateway represent the gateway namespace of an exit node
type Gateway struct {
	prefixZero *net.IPNet
	allocnr    int
	exitnodenr int
}

// New creates a new Gateway object
func New(prefixZero *net.IPNet, allocNr, exitnodenr int) *Gateway {
	return &Gateway{
		prefixZero: prefixZero,
		allocnr:    allocNr,
		exitnodenr: exitnodenr,
	}
}

//Create create the gateway network namespace and configure its default routes and addresses
func (gw *Gateway) Create() error {
	// TODO:
	// if nr.NodeID.ReachabilityV6 == modules.ReachabilityV6ULA {
	// 	return fmt.Errorf("cannot configure an exit point in a hidden node")
	// }

	gwNW, err := ensureGatewayNS(types.GatewayNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to ensure gateway namespace")
	}
	defer gwNW.Close()

	if err := gw.createMacVlan(gwNW); err != nil {
		return err
	}

	if err := gw.createRoutingBridge(gwNW); err != nil {
		return err
	}

	return applyFirewall()
}

func (gw *Gateway) createMacVlan(gwNetNS ns.NetNS) error {
	gwPubName := zosip.GWPubName(gw.exitnodenr, gw.allocnr)
	if macvlan.Exists(gwPubName, gwNetNS) {
		return nil
	}

	pubIface, err := getPublicIface()
	if err != nil {
		return err
	}

	macPubIface, err := macvlan.Create(gwPubName, pubIface, gwNetNS)
	if err != nil {
		return err
	}

	ips := []*net.IPNet{
		zosip.GWPubLL(gw.exitnodenr),
		zosip.GWPubIP6(gw.prefixZero.IP, gw.exitnodenr),
		// zosip.GWPubIP4(), TODO:
	}
	routes := []*netlink.Route{
		{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        net.ParseIP("fe80::1"),
			LinkIndex: macPubIface.Attrs().Index,
		},
	}
	if err := macvlan.Install(macPubIface, ips, routes, gwNetNS); err != nil {
		return errors.Wrap(err, "failed to configure gateway public macvlan")
	}

	return nil
}
func (gw *Gateway) createRoutingBridge(gwNetNS ns.NetNS) error {
	if bridge.Exists(BridgeGateway) {
		return nil
	}

	br, err := bridge.New(BridgeGateway)
	if err != nil {
		return err
	}

	if _, _, err = ip.SetupVethWithName(vethBrSide, vethGWSide, 1500, gwNetNS); err != nil {
		return errors.Wrap(err, "failed to create veth pair for bridge gateway")
	}
	log.Info().
		Str("gatway side", vethGWSide).
		Str("bridge side", vethBrSide).
		Msg("veth pair for ipv4 gateway bridge created")

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", BridgeGateway), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", BridgeGateway)
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", vethBrSide), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on veth pair %s", vethBrSide)
	}

	lVethBr, err := netlink.LinkByName(vethBrSide)
	if err != nil {
		return err
	}

	if err := bridge.AttachNic(lVethBr, br); err != nil {
		return err
	}

	return gwNetNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", vethGWSide), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on veth pair %s", vethGWSide)
		}

		lVethGW, err := netlink.LinkByName(vethGWSide)
		if err != nil {
			return err
		}

		_, ipnet, err := net.ParseCIDR("10.1.0.1/16")
		if err != nil {
			return err
		}
		return netlink.AddrAdd(lVethGW, &netlink.Addr{
			IPNet: ipnet,
		})
	})
}

func applyFirewall() error {
	buf := bytes.Buffer{}

	if err := fwTmpl.Execute(&buf, nil); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, types.GatewayNamespace); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

// ensureGatewayNS check if the gateway namespace exists and if not creates it
// it return the network namespace handle.
// The called MUST close the handle once it is done with it
func ensureGatewayNS(name string) (ns.NetNS, error) {
	var (
		netNS ns.NetNS
		err   error
	)

	netNS, err = namespace.GetByName(name)
	if err != nil {
		log.Info().Msg("create gateway namespace")
		netNS, err = namespace.Create(name)
		if err := netNS.Do(func(_ ns.NetNS) error {
			if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
				return errors.Wrapf(err, "failed to enable ipv6 forwarding in gateway namespace")
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gateway namespace")
	}

	return netNS, nil
}

// AddNetResource adds the routes of a network resource to the gateway network namespace
func (gw *Gateway) AddNetResource(netRes *nr.NetResource) error {
	log.Info().Msg("add network resource to gateway namespace")
	gwNS, err := namespace.GetByName(types.GatewayNamespace)
	if err != nil {
		return errors.Wrap(err, "gateway namespace not found")
	}
	defer gwNS.Close()

	netResNS, err := namespace.GetByName(netRes.NamespaceName())
	if err != nil {
		return errors.Wrapf(err, "namespace %s not found", netRes.NamespaceName())
	}
	defer netResNS.Close()

	if err := gw.configNRIPv6(netRes, gwNS, netResNS); err != nil {
		return err
	}

	if err := gw.configNRIPv4(netRes, gwNS, netResNS); err != nil {
		return err
	}

	return nil
}

func (gw *Gateway) configNRIPv6(netRes *nr.NetResource, gwNS, netResNS ns.NetNS) error {

	epName := netRes.Nibble().EPPubName()
	gwName := netRes.Nibble().GWtoEPName()

	if !ifaceutil.Exists(epName, netResNS) || !ifaceutil.Exists(gwName, gwNS) {
		log.Warn().Msg("one side of the gateway veth pair does not exists, deleting both")
		if err := ifaceutil.Delete(epName, netResNS); err != nil {
			return err
		}
		if err := ifaceutil.Delete(gwName, gwNS); err != nil {
			return err
		}

		log.Info().
			Str("gateway side", gwName).
			Str("exit point side", epName).
			Msg("create a veth pair in the host namespace and send one side into the gateway namespace")
		if _, _, err := ip.SetupVethWithName(epName, gwName, 1500, gwNS); err != nil {
			return errors.Wrap(err, "failed to create veth pair for gateway namespace")
		}
		log.Info().
			Str("gwVeth", gwName).
			Str("epVeth", epName).
			Msg("veth pair for gateway and exit point created")

		// send the other side inside in the exit point network resource namespace
		EPLink, err := netlink.LinkByName(epName)
		if err != nil {
			return errors.Wrapf(err, "failed to get interface %s", epName)
		}
		if err = netlink.LinkSetNsFd(EPLink, int(netResNS.Fd())); err != nil {
			return fmt.Errorf("failed to move interface %s to network resource netns: %v", epName, err)
		}
	}

	err := gwNS.Do(func(_ ns.NetNS) error {
		GWLink, err := netlink.LinkByName(gwName)
		if err != nil {
			return errors.Wrapf(err, "failed to get interface %s in gateway namespace", gwName)
		}

		addr := &netlink.Addr{IPNet: netRes.Nibble().GWtoEPLL(), Label: ""}
		if err := netlink.AddrAdd(GWLink, addr); err != nil && !os.IsExist(err) {
			return err
		}

		route := netRes.Nibble().GWDefaultRoute()
		route.LinkIndex = GWLink.Attrs().Index
		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
			return err
		}

		routes, err := netRes.GWTNRoutesIPv6()
		if err != nil {
			return err
		}
		for _, route := range routes {
			route.LinkIndex = GWLink.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
				return errors.Wrapf(err, "failed to set route %s on %s", route.String(), GWLink.Attrs().Name)
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to configure veth pair end in gateway namespace")
	}

	// configure veth pair inside the exit point network resource namespace
	err = netResNS.Do(func(_ ns.NetNS) error {
		EPLink, err := netlink.LinkByName(epName)
		if err != nil {
			return errors.Wrapf(err, "failed to get interface %s in exit point namespace", epName)
		}

		addr := &netlink.Addr{IPNet: netRes.Nibble().EPPubLL(), Label: ""}
		if err := netlink.AddrAdd(EPLink, addr); err != nil && !os.IsExist(err) {
			return err
		}

		if err := netlink.LinkSetUp(EPLink); err != nil {
			return err
		}

		route := netRes.Nibble().NRDefaultRoute()
		route.LinkIndex = EPLink.Attrs().Index
		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to configure veth pair end in exit point namespace")
	}
	return nil
}

func (gw *Gateway) configNRIPv4(netRes *nr.NetResource, gwNS, netResNS ns.NetNS) error {
	ep4pubName := netRes.Nibble().EP4PubName()
	br4pubName := netRes.Nibble().Br4PubName()

	if !ifaceutil.Exists(ep4pubName, netResNS) || !ifaceutil.Exists(br4pubName, nil) {
		log.Warn().Msg("one side of the gateway veth pair does not exists, deleting both")
		if err := ifaceutil.Delete(ep4pubName, netResNS); err != nil {
			return err
		}
		if err := ifaceutil.Delete(br4pubName, nil); err != nil {
			return err
		}

		if _, _, err := ip.SetupVethWithName(br4pubName, ep4pubName, 1500, netResNS); err != nil {
			return errors.Wrap(err, "failed to create veth pair for bridge gateway")
		}
		log.Info().
			Str("gatway side", br4pubName).
			Str("bridge side", ep4pubName).
			Msg("veth pair for ipv4 gateway bridge created")
	}

	br, err := bridge.Get(BridgeGateway)
	if err != nil {
		return err
	}

	lVethBr, err := netlink.LinkByName(br4pubName)
	if err != nil {
		return err
	}

	if err := bridge.AttachNic(lVethBr, br); err != nil {
		return err
	}

	if err := netResNS.Do(func(_ ns.NetNS) error {
		lep4pub, err := netlink.LinkByName(ep4pubName)
		if err != nil {
			return err
		}

		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", ep4pubName), "1"); err != nil {
			return err
		}

		addr := &netlink.Addr{IPNet: netRes.Nibble().WGIP4()}
		log.Debug().Msgf("set addr %s on %s", addr.IPNet.String(), ep4pubName)
		if err := netlink.AddrAdd(lep4pub, addr); err != nil && !os.IsExist(err) {
			return err
		}

		if err := netlink.LinkSetUp(lep4pub); err != nil {
			return err
		}

		route := &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        net.ParseIP("10.1.0.1"),
			LinkIndex: lep4pub.Attrs().Index,
		}

		log.Debug().Msgf("set route %s on %s %d", route.String(), ep4pubName, lep4pub.Attrs().Index)
		if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return gwNS.Do(func(_ ns.NetNS) error {
		l, err := netlink.LinkByName(vethGWSide)
		if err != nil {
			return err
		}

		routes, err := netRes.GWTNRoutesIPv4()
		if err != nil {
			return err
		}

		for _, route := range routes {
			route.LinkIndex = l.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
				return errors.Wrapf(err, "failed to set route %s on %s", route.String(), l.Attrs().Name)
			}
		}

		// TODO: add IPv4 public IP once farmer can specify ipv4 public address in the farm management
		return nil
	})
}
