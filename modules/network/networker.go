package network

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

type networker struct {
	nodeID      modules.NodeID
	storageDir  string
	netResAlloc NetResourceAllocator
}

// NewNetworker create a new modules.Networker that can be used with zbus
func NewNetworker(storageDir string, allocator NetResourceAllocator) modules.Networker {
	return &networker{
		storageDir:  storageDir,
		netResAlloc: allocator,
	}
}

var _ modules.Networker = (*networker)(nil)

// GetNetwork implements modules.Networker interface
func (n *networker) GetNetwork(id string) (*modules.Network, error) {
	// TODO check signature
	return n.netResAlloc.Get(id)
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node: %s", n.nodeID.ID)
	}

	if err := n.createNetworkResource(network); err != nil {
		return err
	}

	peers, routes, err := n.prepareHidden(network)
	if err != nil {
		return err
	}

	if isPublic(localResource.NodeID) {
		pubPeers, pubRoutes, err := n.preparePublic(network)
		if err != nil {
			return err
		}
		peers = append(peers, pubPeers...)
		routes = append(routes, pubRoutes...)
	}

	exitPeers, exitRoutes, err := n.prepareNonExitNode(network)
	if err != nil {
		return err
	}

	peers = append(peers, exitPeers...)
	routes = append(routes, exitRoutes...)
	for _, route := range routes {
		fmt.Println(route.String())
	}

	if err := n.configWG(network, peers, routes); err != nil {
		return err
	}
	return nil
}

func (n *networker) DeleteNetResource(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}
	var (
		nibble     = zosip.NewNibble(localResource.Prefix, network.AllocationNR)
		netnsName  = nibble.NetworkName()
		bridgeName = nibble.BridgeName()
	)
	if err := bridge.Delete(bridgeName); err != nil {
		return err
	}
	return namespace.Delete(netnsName)
}

// createNetworkResource creates a network namespace and a bridge
// and a wireguard interface and then move it interface inside
// the net namespace
func (n *networker) createNetworkResource(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}

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

	log.Info().Str("namespace", netnsName).Msg("Create namesapce")
	netns, err := namespace.Create(netnsName)
	if err != nil {
		return err
	}

	hostIface := &current.Interface{}
	var handler = func(hostNS ns.NetNS) error {
		if err := zosip.EnableIPv6Forwarding(); err != nil {
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
		ip, ipnetv4, err := net.ParseCIDR(fmt.Sprintf("10.%d.%d.1/24", a, b))
		if err != nil {
			return err
		}
		ipnetv4.IP = ip

		for _, ipnet := range []*net.IPNet{ipnetv6, ipnetv4} {
			log.Info().Str("addr", ipnet.String()).Msg("set address on veth interface")
			addr := &netlink.Addr{IPNet: ipnet, Label: ""}
			if err = netlink.AddrAdd(link, addr); err != nil {
				return err
			}
		}

		return nil
	}
	if err := netns.Do(handler); err != nil {
		return err
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return err
	}

	log.Info().
		Str("veth", vethName).
		Str("bridge", bridgeName).
		Msg("attach veth to bridge")
	if err := bridge.AttachNic(hostVeth, br); err != nil {
		return err
	}

	log.Info().Str("wg", wgName).Msg("create wireguard interface")
	wg, err := wireguard.New(wgName)
	if err != nil {
		return err
	}

	log.Info().
		Str("wg", wgName).
		Str("namespace", netnsName).
		Msg("move wireguard into network namespace")
	if err := namespace.SetLink(wg, netnsName); err != nil {
		return err
	}

	return nil
}

func (n *networker) configureExitNetNR(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}

	hiddenPrefixes := hiddenPrefixes(network.Resources)
	peers := make([]wireguard.Peer, 0, len(hiddenPrefixes))
	routes := make([]*netlink.Route, 0, len(hiddenPrefixes))

	for _, peer := range localResource.Peers {
		if peer.Type != modules.ConnTypeWireguard {
			continue
		}
		if peer.Prefix.String() == localResource.Prefix.String() {
			continue
		}
		if !isIn(peer.Prefix.String(), hiddenPrefixes) {
			continue
		}

		nibble := zosip.NewNibble(peer.Prefix, network.AllocationNR)
		a, b, err := nibble.ToV4()
		if err != nil {
			return err
		}

		peers = append(peers, wireguard.Peer{
			PublicKey: peer.Connection.Key,
			AllowedIPs: []string{
				fmt.Sprintf("fe80::%s/128", nibble.Hex()),
				fmt.Sprintf("172.16.%d.%d/32", a, b),
				peer.Prefix.String(),
			},
		})
		routes = append(routes, &netlink.Route{
			Dst: peer.Prefix,
			Gw:  net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
		})
	}

	localNibble := zosip.NewNibble(localResource.Prefix, network.AllocationNR)
	netns, err := namespace.GetByName(localNibble.NetworkName())
	if err != nil {
		return err
	}
	storagePath := filepath.Join(n.storageDir, localNibble.Hex())
	var key wgtypes.Key
	key, err = wireguard.LoadKey(storagePath)
	if err != nil {
		key, err = wireguard.GenerateKey(storagePath)
		if err != nil {
			return err
		}
	}

	var handler = func(_ ns.NetNS) error {

		wg, err := wireguard.GetByName(localNibble.WiregardName())
		if err != nil {
			return err
		}

		if err := wg.SetAddr(localResource.LinkLocal.String()); err != nil {
			return err
		}
		a, b, err := localNibble.ToV4()
		if err != nil {
			return err
		}
		if err := wg.SetAddr(fmt.Sprintf("172.16.%d.%d/16", a, b)); err != nil {
			return err
		}

		log.Info().Msg("configure wireguard interface")
		if err = wg.Configure(key.String(), peers); err != nil {
			return err
		}

		for _, route := range routes {
			route.LinkIndex = wg.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil {
				log.Error().
					Err(err).
					Str("route", route.String()).
					Msg("fail to set route")
				return err
			}
		}

		return nil
	}
	return netns.Do(handler)
}

func (n *networker) prepareHidden(network *modules.Network) ([]wireguard.Peer, []*netlink.Route, error) {
	var (
		publicPrefixes = publicPrefixes(network.Resources)
	)

	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return nil, nil, fmt.Errorf("not network resource for this node")
	}

	peers := make([]wireguard.Peer, 0, len(publicPrefixes)+1)
	routes := make([]*netlink.Route, 0, len(publicPrefixes))

	for _, peer := range localResource.Peers {
		if peer.Type != modules.ConnTypeWireguard {
			continue
		}
		if peer.Prefix.String() == localResource.Prefix.String() {
			continue
		}

		nibble := zosip.NewNibble(peer.Prefix, network.AllocationNR)
		a, b, err := nibble.ToV4()
		if err != nil {
			return nil, nil, err
		}

		if isIn(peer.Prefix.String(), publicPrefixes) {
			peers = append(peers, wireguard.Peer{
				PublicKey: peer.Connection.Key,
				Endpoint:  endpoint(peer),
				AllowedIPs: []string{
					fmt.Sprintf("fe80::%s/128", nibble.Hex()),
					fmt.Sprintf("172.16.%d.%d/32", a, b),
					peer.Prefix.String(),
				},
			})
			routes = append(routes, &netlink.Route{
				Dst: peer.Prefix,
				Gw:  net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
			})
		}
	}
	return peers, routes, nil
}

func (n *networker) preparePublic(network *modules.Network) ([]wireguard.Peer, []*netlink.Route, error) {
	var (
		publicPrefixes = publicPrefixes(network.Resources)
	)

	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return nil, nil, fmt.Errorf("not network resource for this node")
	}

	peers := make([]wireguard.Peer, 0, len(publicPrefixes)+1)
	routes := make([]*netlink.Route, 0, len(publicPrefixes))

	// we are a public node
	for _, peer := range localResource.Peers {
		if peer.Type != modules.ConnTypeWireguard {
			continue
		}
		if peer.Prefix.String() == localResource.Prefix.String() {
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
				fmt.Sprintf("172.16.%d.%d/32", a, b),
				peer.Prefix.String(),
			},
		}

		if isIn(peer.Prefix.String(), publicPrefixes) {
			wgPeer.Endpoint = endpoint(peer)
		}
		peers = append(peers, wgPeer)

		if peer.Prefix.String() == network.Exit.Prefix.String() {
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

func (n *networker) prepareNonExitNode(network *modules.Network) ([]wireguard.Peer, []*netlink.Route, error) {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return nil, nil, fmt.Errorf("not network resource for this node")
	}

	peers := make([]wireguard.Peer, 0)
	routes := make([]*netlink.Route, 0)

	// add exit node to the list of peers
	exitPeer, err := getPeer(network.Exit.Prefix.String(), localResource.Peers)
	if err != nil {
		return nil, nil, err
	}
	peers = append(peers, wireguard.Peer{
		PublicKey: exitPeer.Connection.Key,
		Endpoint:  endpoint(exitPeer),
		AllowedIPs: []string{
			"0.0.0.0/0",
			"::/0",
		},
	})
	nibble := zosip.NewNibble(exitPeer.Prefix, network.AllocationNR)
	// if we are not the exit node, then add the default route to the exit node
	if localResource.Prefix.String() != network.Exit.Prefix.String() {
		dst := &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(64, 128),
		}
		routes = append(routes, &netlink.Route{
			Dst: dst,
			Gw:  net.ParseIP(fmt.Sprintf("fe80::%s", nibble.Hex())),
		})

		a, b, err := nibble.ToV4()
		if err != nil {
			return nil, nil, err
		}
		dst = &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("10.%d.%d.0", a, b)),
			Mask: net.CIDRMask(24, 32),
		}
		routes = append(routes, &netlink.Route{
			Dst: dst,
			Gw:  net.ParseIP(fmt.Sprintf("172.16.%d.%d", a, b)),
		})

		dst = &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		}
		routes = append(routes, &netlink.Route{
			Dst: dst,
			Gw:  net.ParseIP(fmt.Sprintf("172.16.%d.%d", a, b)),
		})
	}

	return peers, routes, nil
}

func (n *networker) configWG(network *modules.Network, wgPeers []wireguard.Peer, routes []*netlink.Route) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}

	localNibble := zosip.NewNibble(localResource.Prefix, network.AllocationNR)
	netns, err := namespace.GetByName(localNibble.NetworkName())
	if err != nil {
		return err
	}
	storagePath := filepath.Join(n.storageDir, localNibble.Hex())
	var key wgtypes.Key
	key, err = wireguard.LoadKey(storagePath)
	if err != nil {
		key, err = wireguard.GenerateKey(storagePath)
		if err != nil {
			return err
		}
	}

	var handler = func(_ ns.NetNS) error {

		wg, err := wireguard.GetByName(localNibble.WiregardName())
		if err != nil {
			return err
		}

		if err := wg.SetAddr(localResource.LinkLocal.String()); err != nil {
			return err
		}
		a, b, err := localNibble.ToV4()
		if err != nil {
			return err
		}
		if err := wg.SetAddr(fmt.Sprintf("172.16.%d.%d/16", a, b)); err != nil {
			return err
		}

		log.Info().Msg("configure wireguard interface")
		if err = wg.Configure(key.String(), wgPeers); err != nil {
			return err
		}

		for _, route := range routes {
			route.LinkIndex = wg.Attrs().Index
			if err := netlink.RouteAdd(route); err != nil {
				log.Error().
					Err(err).
					Str("route", route.String()).
					Msg("fail to set route")
				return err
			}
		}

		return nil
	}
	return netns.Do(handler)
}

// localResource return the net resource of the local node from a list of net resources
func (n *networker) localResource(resources []*modules.NetResource) *modules.NetResource {
	for _, resource := range resources {
		if resource.NodeID.ID == n.nodeID.ID {
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

func isPublic(nodeID modules.NodeID) bool {
	return nodeID.ReachabilityV6 == modules.ReachabilityV6Public ||
		nodeID.ReachabilityV4 == modules.ReachabilityV4Public
}

func isHidden(nodeID modules.NodeID) bool {
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

func wgIP(prefix net.IPNet) (*net.IPNet, error) {
	prefixIP := []byte(prefix.IP.To16())
	id := prefixIP[6:8]
	_, ipnet, err := net.ParseCIDR(fmt.Sprintf("fe80::%x/64", id))
	if err != nil {
		return nil, err
	}
	return ipnet, nil
}
