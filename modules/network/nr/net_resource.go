package nr

import (
	"bytes"
	"fmt"
	"net"
	"os"

	mapset "github.com/deckarep/golang-set"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/nft"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

// NetResource holds the logic to configure an network resource
type NetResource struct {
	resource       *modules.NetResource
	exit           *modules.NetResource
	networkID      modules.NetID
	publicPrefixes []string
	allocNr        int8
	nibble         *zosip.Nibble

	privateKey wgtypes.Key
}

// New creates a new NetResource object
func New(nodeID string, network *modules.Network, privateKey wgtypes.Key) (*NetResource, error) {
	var err error
	nr := &NetResource{
		networkID:      network.NetID,
		allocNr:        network.AllocationNR,
		privateKey:     privateKey,
		publicPrefixes: publicPrefixes(network.Resources),
	}

	nr.resource, err = ResourceByNodeID(nodeID, network.Resources)
	if err != nil {
		return nil, err
	}

	nr.exit, err = exitResource(network.Resources)
	if err != nil {
		return nil, err
	}
	nr.nibble, err = zosip.NewNibble(nr.resource.Prefix, nr.allocNr)
	if err != nil {
		return nil, err
	}

	return nr, nil
}

// Nibble returns the nibble object of the Network Resource
func (nr *NetResource) Nibble() *zosip.Nibble {
	return nr.nibble
}

// IsExit returns the exitNode number and true if this network resource holds
// the exit point of the network
// otherwise it returns 0 and false
func (nr *NetResource) IsExit() (int, bool) {
	return nr.resource.ExitPoint, nr.resource.ExitPoint > 0
}

// NamespaceName returns the name of the network resource namespace
func (nr *NetResource) NamespaceName() string {
	return nr.nibble.NamespaceName()
}

// Create setup the basic components of the network resource
// network namespace, bridge, wireguard interface and veth pair
func (nr *NetResource) Create(pubNS ns.NetNS) error {
	if err := nr.createBridge(); err != nil {
		return err
	}
	if err := nr.createNetNS(); err != nil {
		return err
	}
	if err := nr.createVethPair(); err != nil {
		return err
	}
	if err := nr.createWireguard(pubNS); err != nil {
		return err
	}

	if err := nr.applyFirewall(); err != nil {
		return err
	}

	return nil
}

// Configure sets the routes and IP addresses on the
// wireguard interface of the network resources
func (nr *NetResource) Configure() error {
	routes, err := nr.routes()
	if err != nil {
		return errors.Wrap(err, "failed to generate routes for wireguard")
	}

	wgPeers, err := nr.wgPeers()
	if err != nil {
		return errors.Wrap(err, "failed to wireguard peer configuration")
	}

	nsName := nr.nibble.NamespaceName()
	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return fmt.Errorf("network namespace %s does not exits", nsName)
	}

	var handler = func(_ ns.NetNS) error {

		wg, err := wireguard.GetByName(nr.nibble.WGName())
		if err != nil {
			return errors.Wrapf(err, "failed to get wireguard interface %s", nr.nibble.WGName())
		}

		localPeer, err := PeerByPrefix(nr.resource.Prefix.String(), nr.resource.Peers)
		if err != nil {
			return fmt.Errorf("not peer found for local network resource: %s", err)
		}

		if err = wg.Configure(nr.privateKey.String(), int(localPeer.Connection.Port), wgPeers); err != nil {
			return errors.Wrap(err, "failed to configure wireguard interface")
		}

		addrs, err := netlink.AddrList(wg, netlink.FAMILY_ALL)
		if err != nil {
			return err
		}
		curAddrs := mapset.NewSet()
		for _, addr := range addrs {
			curAddrs.Add(addr.IPNet.String())
		}

		newAddrs := mapset.NewSet()
		newAddrs.Add(nr.resource.LinkLocal.String())
		//TODO: move this hack into nibble method
		ipnet := nr.nibble.WGAllowedIP4()
		ipnet.Mask = net.CIDRMask(16, 32)
		newAddrs.Add(ipnet.String())
		toRemove := curAddrs.Difference(newAddrs)
		toAdd := newAddrs.Difference(curAddrs)

		log.Info().Msgf("current %s", curAddrs.String())
		log.Info().Msgf("to add %s", toAdd.String())
		log.Info().Msgf("to remove %s", toRemove.String())

		for addr := range toAdd.Iter() {
			addr, _ := addr.(string)
			log.Debug().Str("ip", addr).Msg("set ip on wireguard interface")
			if err := wg.SetAddr(addr); err != nil {
				return errors.Wrapf(err, "failed to set address %s on wireguard interface %s", addr, wg.Attrs().Name)
			}
		}

		for addr := range toRemove.Iter() {
			addr, _ := addr.(string)
			log.Debug().Str("ip", addr).Msg("unset ip on wireguard interface")
			// TODO: zaibon
			// if err := wg.UsetAddr(addr); err != nil {
			// 	return errors.Wrapf(err, "failed to set address %s on wireguard interface %s", addr, wg.Attrs().Name)
			// }
		}

		for _, route := range routes {
			route.LinkIndex = wg.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
				log.Error().
					Err(err).
					Str("route", route.String()).
					Msg("fail to set route")
				return errors.Wrapf(err, "failed to add route %s", route.String())
			}
		}

		return nil
	}
	return netNS.Do(handler)
}

// Delete removes all the interfaces and namespaces created by the Create method
func (nr *NetResource) Delete() error {
	netnsName := nr.nibble.NamespaceName()
	bridgeName := nr.nibble.BridgeName()

	if bridge.Exists(bridgeName) {
		if err := bridge.Delete(bridgeName); err != nil {
			log.Error().
				Err(err).
				Str("bridge", bridgeName).
				Msg("failed to delete network resource bridge")
			return err
		}
	}

	if namespace.Exists(netnsName) {
		netResNS, err := namespace.GetByName(netnsName)
		if err != nil {
			return err
		}
		// don't explicitly netResNS.Close() the netResNS here, namespace.Delete will take care of it
		if err := namespace.Delete(netResNS); err != nil {
			log.Error().
				Err(err).
				Str("namespace", netnsName).
				Msg("failed to delete network resource namespace")
		}
	}

	return nil
}

func (nr *NetResource) routes() ([]*netlink.Route, error) {
	routes := make([]*netlink.Route, 0, len(nr.publicPrefixes))

	for _, peer := range nr.resource.Peers {

		prefixStr := peer.Prefix.String()
		if peer.Type != modules.ConnTypeWireguard || // wireguard is the only supported connection type at the moment
			prefixStr == nr.resource.Prefix.String() { // skip myself
			continue
		}

		nibble, err := zosip.NewNibble(peer.Prefix, nr.allocNr)
		if err != nil {
			return nil, err
		}

		routes = append(routes, &netlink.Route{
			Dst: peer.Prefix,
			Gw:  net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
		})

		//TODO: move this hack into nibble method
		ipnet := nibble.NRLocalIP4()
		ipnet.IP[15] = 0x00
		routes = append(routes, &netlink.Route{
			Dst: ipnet,
			Gw:  nibble.WGIP4RT().IP,
		})

		if prefixStr == nr.exit.Prefix.String() { //special configuration for exit point
			// add default ipv6 route to the exit node
			routes = append(routes, nibble.RouteIPv6Exit())
			// add ipv4 route to the exit node
			routes = append(routes, nibble.RouteIPv4Exit())
			// add default ipv4 route to the exit node
			routes = append(routes, nibble.RouteIPv4DefaultExit())
		}
	}

	return routes, nil
}

func (nr *NetResource) wgPeers() ([]*wireguard.Peer, error) {
	exitPeer, err := PeerByPrefix(nr.exit.Prefix.String(), nr.resource.Peers)
	if err != nil {
		return nil, err
	}

	wgPeers := make([]*wireguard.Peer, 0, len(nr.publicPrefixes)+1)

	for _, peer := range nr.resource.Peers {
		prefixStr := peer.Prefix.String()
		if peer.Type != modules.ConnTypeWireguard || // wireguard is the only supported connection type at the moment
			prefixStr == nr.resource.Prefix.String() { // skip myself
			continue
		}

		nibble, err := zosip.NewNibble(peer.Prefix, nr.allocNr)
		if err != nil {
			return nil, err
		}

		if prefixStr != nr.exit.Prefix.String() {
			wgPeer := &wireguard.Peer{
				PublicKey: peer.Connection.Key,
				AllowedIPs: []string{
					nibble.WGAllowedIP6().String(),
					nibble.WGAllowedIP4().String(),
					nibble.WGAllowedIP4Net().String(),
					peer.Prefix.String(),
				},
			}

			if isIn(peer.Prefix.String(), nr.publicPrefixes) {
				wgPeer.Endpoint = zosip.WGEndpoint(peer)
			}

			log.Info().Str("peer prefix", peer.Prefix.String()).Msg("generate wireguard configuration for peer")
			wgPeers = append(wgPeers, wgPeer)

		} else { //special configuration for exit peer
			// add wg peer config to the exit node
			allowedIPs := zosip.WGExitPeerAllowIPs()
			wgPeers = append(wgPeers, &wireguard.Peer{
				PublicKey: exitPeer.Connection.Key,
				Endpoint:  zosip.WGEndpoint(exitPeer),
				AllowedIPs: []string{
					allowedIPs[0].String(),
					allowedIPs[1].String(),
				},
			})
		}
	}

	return wgPeers, nil
}

// GWTNRoutesIPv6 returns the routes to set in the gateway network namespace
// to be abe to reach a network resource
func (nr *NetResource) GWTNRoutesIPv6() ([]*netlink.Route, error) {
	routes := make([]*netlink.Route, 0)

	for _, peer := range nr.resource.Peers {
		routes = append(routes, &netlink.Route{
			Dst: peer.Prefix,
			Gw:  nr.nibble.EPPubLL().IP,
		})
	}

	return routes, nil
}

// GWTNRoutesIPv4 returns the routes to set in the gateway network namespace
// to be abe to reach a network resource
func (nr *NetResource) GWTNRoutesIPv4() ([]*netlink.Route, error) {
	routes := make([]*netlink.Route, 0)

	for _, peer := range nr.resource.Peers {
		peerNibble, err := zosip.NewNibble(peer.Prefix, nr.allocNr)
		if err != nil {
			return nil, err
		}

		//IPv4 routing
		ipnet := peerNibble.WGIP4RT()
		ipnet.Mask = net.CIDRMask(32, 32)
		routes = append(routes, &netlink.Route{
			Dst: ipnet,
			Gw:  nr.nibble.EPPubIP4R().IP,
		})
		// IPv4 wiregard host routing
		ipnet = peerNibble.NRLocalIP4()
		ipnet.IP[15] = 0x00

		routes = append(routes, &netlink.Route{
			Dst: ipnet,
			Gw:  nr.nibble.EPPubIP4R().IP,
		})
	}

	return routes, nil
}

func (nr *NetResource) createNetNS() error {
	netnsName := nr.nibble.NamespaceName()

	if namespace.Exists(netnsName) {
		return nil
	}

	log.Info().Str("namespace", netnsName).Msg("Create namespace")

	netNS, err := namespace.Create(netnsName)
	if err != nil {
		return err
	}
	netNS.Close()
	return nil
}

// createVethPair creates a veth pair inside the namespace of the
// network resource and sends one side back to the host namespace
// the host side of the veth pair is attach to the bridge of the network resource
func (nr *NetResource) createVethPair() error {

	nsName := nr.nibble.NamespaceName()
	vethName := nr.nibble.NRLocalName()
	bridgeName := nr.nibble.BridgeName()

	// check if the veth already exists
	if _, err := netlink.LinkByName(vethName); err == nil {
		return nil
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return fmt.Errorf("network namespace %s does not exits", nsName)
	}
	defer netNS.Close()

	brLink, err := bridge.Get(bridgeName)
	if err != nil {
		return fmt.Errorf("bridge %s does not exits", bridgeName)
	}

	exists := false
	hostIface := ""
	var handler = func(hostNS ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return err
		}

		if _, err := netlink.LinkByName(vethName); err == nil {
			exists = true
			return nil
		}

		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}

		log.Info().
			Str("namespace", nsName).
			Str("veth", vethName).
			Msg("Create veth pair in net namespace")

		hostVeth, containerVeth, err := ip.SetupVeth(vethName, 1500, hostNS)
		if err != nil {
			return err
		}
		hostIface = hostVeth.Name

		link, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		for _, ipnet := range []*net.IPNet{nr.resource.Prefix, nr.nibble.NRLocalIP4()} {
			log.Info().Str("addr", ipnet.String()).Msg("set address on veth interface")
			addr := &netlink.Addr{IPNet: ipnet, Label: ""}
			if err = netlink.AddrAdd(link, addr); err != nil && !os.IsExist(err) {
				return err
			}
		}

		return nil
	}
	if err := netNS.Do(handler); err != nil {
		return err
	}

	if exists {
		return nil
	}

	hostVeth, err := netlink.LinkByName(hostIface)
	if err != nil {
		return err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", hostVeth.Attrs().Name), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on veth %s", hostVeth.Attrs().Name)
	}

	log.Info().
		Str("veth", vethName).
		Str("bridge", bridgeName).
		Msg("attach veth to bridge")

	return bridge.AttachNic(hostVeth, brLink)
}

// createBridge creates a bridge in the host namespace
// this bridge is used to connect containers from the name network
func (nr *NetResource) createBridge() error {
	name := nr.nibble.BridgeName()

	if bridge.Exists(name) {
		return nil
	}

	log.Info().Str("bridge", name).Msg("Create bridge")

	_, err := bridge.New(name)
	if err != nil {
		return err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}
	return nil
}

// createWireguard the wireguard interface of the network resource
// if a public namespace handle (pubNetNS) is not nil, the interface is created
// inside the public namespace and then moved inside the network resource namespace
func (nr *NetResource) createWireguard(pubNetNS ns.NetNS) error {
	wgName := nr.nibble.WGName()
	nsName := nr.nibble.NamespaceName()
	nrNetNS, err := namespace.GetByName(nsName)
	if err != nil {
		return err
	}
	defer nrNetNS.Close()

	exists := false
	nrNetNS.Do(func(hostNS ns.NetNS) error {
		_, err := netlink.LinkByName(wgName)
		if err == nil {
			exists = true
		}
		return nil
	})

	// wireguard already exist, early exit
	if exists {
		return nil
	}

	slog := log.With().
		Str("wg", wgName).
		Str("namespace", nsName).
		Logger()

	if pubNetNS == nil {
		// create the wg interface in the host network namespace
		log.Info().Str("wg", wgName).Msg("create wireguard interface in host namespace")
		_, err := wireguard.New(wgName)
		if err != nil {
			return err
		}
	} else {
		if err := pubNetNS.Do(func(hostNS ns.NetNS) error {

			log.Info().Str("wg", wgName).Msg("create wireguard interface in public namespace")
			wg, err := wireguard.New(wgName)
			if err != nil {
				return err
			}

			slog.Info().Msg("move wireguard into host namespace")
			return netlink.LinkSetNsFd(wg, int(hostNS.Fd()))
		}); err != nil {
			return err
		}
	}

	slog.Info().Msg("move wireguard into net resource namespace")
	wg, err := netlink.LinkByName(wgName)
	if err != nil {
		return err
	}

	return netlink.LinkSetNsFd(wg, int(nrNetNS.Fd()))
}

func (nr *NetResource) applyFirewall() error {
	netnsName := nr.nibble.NamespaceName()

	data := struct {
		Iifname string
	}{
		nr.nibble.EP4PubName(),
	}
	buf := bytes.Buffer{}

	if err := fwTmpl.Execute(&buf, data); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, netnsName); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

func isIn(target string, l []string) bool {
	for _, x := range l {
		if target == x {
			return true
		}
	}
	return false
}

func exitResource(r []*modules.NetResource) (*modules.NetResource, error) {
	for _, res := range r {
		if res.ExitPoint > 0 {
			return res, nil
		}
	}
	return nil, fmt.Errorf("not net resource with exit flag enabled found")
}

func publicPrefixes(resources []*modules.NetResource) []string {
	output := []string{}
	for _, res := range resources {
		if isPublic(res.NodeID) {
			output = append(output, res.Prefix.String())
		}
	}
	return output
}

func isPublic(nodeID *modules.NodeID) bool {
	return nodeID.ReachabilityV6 == modules.ReachabilityV6Public ||
		nodeID.ReachabilityV4 == modules.ReachabilityV4Public
}

func isHidden(nodeID *modules.NodeID) bool {
	return nodeID.ReachabilityV6 == modules.ReachabilityV6ULA ||
		nodeID.ReachabilityV4 == modules.ReachabilityV4Hidden
}
