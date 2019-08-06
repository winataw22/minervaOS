package network

import (
	"fmt"
	"net"
	"syscall"

	mapset "github.com/deckarep/golang-set"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

// createNetworkResource creates a network namespace and a bridge
// and a wireguard interface and then move it interface inside
// the net namespace
func createNetworkResource(localResource *modules.NetResource, network *modules.Network) error {
	var (
		nibble     = zosip.NewNibble(localResource.Prefix, network.AllocationNR)
		netnsName  = nibble.NetworkName()
		bridgeName = nibble.BridgeName()
		wgName     = nibble.WiregardName()
		vethName   = nibble.VethName()
	)

	log.Info().Str("bridge", bridgeName).Msg("Create bridge")
	br, err := bridge.New(bridgeName)
	if err != nil {
		return err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", bridgeName), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", bridgeName)
	}

	log.Info().Str("namespace", netnsName).Msg("Create namesapce")
	netResNS, err := namespace.Create(netnsName)
	if err != nil {
		return err
	}
	defer func() {
		if err := netResNS.Close(); err != nil {
			panic(err)
		}
	}()

	hostIface := &current.Interface{}
	var handler = func(hostNS ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return err
		}

		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}

		log.Info().
			Str("namespace", netnsName).
			Str("veth", vethName).
			Msg("Create veth pair in net namespace")
		hostVeth, containerVeth, err := ip.SetupVeth(vethName, 1500, hostNS)
		if err != nil {
			return err
		}
		hostIface.Name = hostVeth.Name

		link, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		ipnetv6 := localResource.Prefix
		a, b, err := nibble.ToV4()
		if err != nil {
			return err
		}
		ipnetv4 := &net.IPNet{
			IP:   net.IPv4(10, a, b, 1),
			Mask: net.CIDRMask(24, 32),
		}

		for _, ipnet := range []*net.IPNet{ipnetv6, ipnetv4} {
			log.Info().Str("addr", ipnet.String()).Msg("set address on veth interface")
			addr := &netlink.Addr{IPNet: ipnet, Label: ""}
			if err = netlink.AddrAdd(link, addr); err != nil {
				return err
			}
		}

		return nil
	}
	if err := netResNS.Do(handler); err != nil {
		return err
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
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
	if err := bridge.AttachNic(hostVeth, br); err != nil {
		return err
	}

	// if there is a public namespace on the node, then
	// we need to create the wireguard interface inside the public namespace then move
	// it into the network resource namespace
	//
	// if there is no public namespace, simply create the wireguard in the host namespace
	// and move it into the network resource namespace

	if namespace.Exists(PublicNamespace) {
		pubNS, err := namespace.GetByName(PublicNamespace)
		if err != nil {
			return err
		}
		defer func() {
			if err := pubNS.Close(); err != nil {
				panic(err)
			}
		}()

		if err = pubNS.Do(func(hostNS ns.NetNS) error {
			log.Info().Str("wg", wgName).Msg("create wireguard interface")
			wg, err := wireguard.New(wgName)
			if err != nil {
				return err
			}
			log.Info().
				Str("wg", wgName).
				Str("namespace", hostNS.Path()).
				Msg("move wireguard into host namespace")

			// move it into the host namespace
			if err := netlink.LinkSetNsFd(wg, int(hostNS.Fd())); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	} else {
		// create the wg interface in the host network namespace
		log.Info().Str("wg", wgName).Msg("create wireguard interface")
		_, err := wireguard.New(wgName)
		if err != nil {
			return err
		}
	}

	log.Info().
		Str("wg", wgName).
		Str("namespace", netnsName).
		Msg("move wireguard into net resource namespace")
	wg, err := netlink.LinkByName(wgName)
	if err != nil {
		return err
	}

	if err := netlink.LinkSetNsFd(wg, int(netResNS.Fd())); err != nil {
		return err
	}

	return nil
}

func genWireguardPeers(localResource *modules.NetResource, network *modules.Network) ([]wireguard.Peer, []*netlink.Route, error) {
	publicPrefixes := publicPrefixes(network.Resources)
	exitNetRes, err := exitResource(network.Resources)
	if err != nil {
		return nil, nil, err
	}

	peers := make([]wireguard.Peer, 0, len(publicPrefixes)+1)
	routes := make([]*netlink.Route, 0, len(publicPrefixes))

	// we are a public node
	for _, peer := range localResource.Peers {
		if peer.Type != modules.ConnTypeWireguard || // wireguard is the only supported connection type at the moment
			peer.Prefix.String() == localResource.Prefix.String() || // skip ourself
			peer.Prefix.String() == exitNetRes.Prefix.String() { // skip exit peer cause we add it in genWireguardExitPeers
			continue
		}

		nibble := zosip.NewNibble(peer.Prefix, network.AllocationNR)
		a, b, err := nibble.ToV4()
		if err != nil {
			return nil, nil, err
		}

		wgPeer := wireguard.Peer{
			PublicKey: peer.Connection.Key,
			AllowedIPs: []string{
				fmt.Sprintf("fe80::%s/128", nibble.Hex()),
				fmt.Sprintf("10.255.%d.%d/32", a, b),
				peer.Prefix.String(),
			},
		}

		if isIn(peer.Prefix.String(), publicPrefixes) {
			wgPeer.Endpoint = endpoint(peer)
		}

		log.Info().Str("peer prefix", peer.Prefix.String()).Msg("generate wireguard configuration for peer")
		peers = append(peers, wgPeer)

		if peer.Prefix.String() == exitNetRes.Prefix.String() {
			// we don't add the route to the exit node here cause it's
			// done in the prepareNonExitNode method
			continue
		}

		routes = append(routes, &netlink.Route{
			Dst: peer.Prefix,
			Gw:  net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
		})
	}

	return peers, routes, nil
}

func genWireguardExitPeers(localResource *modules.NetResource, network *modules.Network) ([]wireguard.Peer, []*netlink.Route, error) {
	var (
		peers      = make([]wireguard.Peer, 0)
		routes     = make([]*netlink.Route, 0)
		exitNetRes *modules.NetResource
		exitPeer   *modules.Peer
		err        error
	)

	exitNetRes, err = exitResource(network.Resources)
	if err != nil {
		return nil, nil, err
	}

	// add exit node to the list of peers
	exitPeer, err = getPeer(exitNetRes.Prefix.String(), localResource.Peers)
	if err != nil {
		return nil, nil, err
	}

	// add wg peer config to the exit node
	peers = append(peers, wireguard.Peer{
		PublicKey: exitPeer.Connection.Key,
		Endpoint:  endpoint(exitPeer),
		AllowedIPs: []string{
			"0.0.0.0/0",
			"::/0",
		},
	})

	// add default ipv6 route to the exit node
	nibble := zosip.NewNibble(exitPeer.Prefix, network.AllocationNR)
	routes = append(routes, &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
	})

	a, b, err := nibble.ToV4()
	if err != nil {
		return nil, nil, err
	}

	// add ipv4 route to the exit network resource
	routes = append(routes, &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("10.%d.%d.0", a, b)),
			Mask: net.CIDRMask(24, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", a, b)),
	})

	// add default ipv4 route to the exit node
	routes = append(routes, &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", a, b)),
	})

	return peers, routes, nil
}

func configWG(localResource *modules.NetResource, network *modules.Network, wgPeers []wireguard.Peer, routes []*netlink.Route, wgKey wgtypes.Key) error {
	localNibble := zosip.NewNibble(localResource.Prefix, network.AllocationNR)
	netns, err := namespace.GetByName(localNibble.NetworkName())
	if err != nil {
		return err
	}
	defer netns.Close()

	var handler = func(_ ns.NetNS) error {

		a, b, err := localNibble.ToV4()
		if err != nil {
			return err
		}

		wg, err := wireguard.GetByName(localNibble.WiregardName())
		if err != nil {
			return errors.Wrapf(err, "failed to get wireguard interface %s", localNibble.WiregardName())
		}

		localPeer, err := getPeer(localResource.Prefix.String(), localResource.Peers)
		if err != nil {
			return fmt.Errorf("not peer found for local network resource: %s", err)
		}

		if err = wg.Configure(wgKey.String(), int(localPeer.Connection.Port), wgPeers); err != nil {
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
		newAddrs.Add(localResource.LinkLocal.String())
		newAddrs.Add(fmt.Sprintf("10.255.%d.%d/16", a, b))

		toRemove := curAddrs.Difference(newAddrs)
		toAdd := newAddrs.Difference(curAddrs)

		log.Info().Msgf("current %s/n", curAddrs.String())
		log.Info().Msgf("to add %s/n", toAdd.String())
		log.Info().Msgf("to remove %s/n", toRemove.String())

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
			// if err := wg.UsetAddr(addr); err != nil {
			// 	return errors.Wrapf(err, "failed to set address %s on wireguard interface %s", addr, wg.Attrs().Name)
			// }
		}

		for _, route := range routes {
			route.LinkIndex = wg.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil && err != syscall.EEXIST {
				log.Error().
					Err(err).
					Str("route", route.String()).
					Msg("fail to set route")
				return errors.Wrapf(err, "failed to add route %s", route.String())
			}
		}

		return nil
	}
	return netns.Do(handler)
}

// localResource return the net resource of the local node from a list of net resources
func (n *networker) localResource(resources []*modules.NetResource) *modules.NetResource {
	for _, resource := range resources {
		if resource.NodeID.ID == n.identity.NodeID().Identity() {
			return resource
		}
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
		if res.ExitPoint {
			return res, nil
		}
	}
	return nil, fmt.Errorf("not net resource with exit flag enabled found")
}

func getPeer(prefix string, peers []*modules.Peer) (*modules.Peer, error) {
	for _, peer := range peers {
		if peer.Prefix.String() == prefix {
			return peer, nil
		}
	}
	return nil, fmt.Errorf("peer not found")
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

func hiddenPrefixes(resources []*modules.NetResource) []string {
	output := []string{}
	for _, res := range resources {
		if isHidden(res.NodeID) {
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

func endpoint(peer *modules.Peer) string {
	var endpoint string
	if peer.Connection.IP.To16() != nil {
		endpoint = fmt.Sprintf("[%s]:%d", peer.Connection.IP.String(), peer.Connection.Port)
	} else {
		endpoint = fmt.Sprintf("%s:%d", peer.Connection.IP.String(), peer.Connection.Port)
	}
	return endpoint
}
