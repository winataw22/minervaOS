package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/provision"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/emicklei/dot"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func cmdGraphNetwork(c *cli.Context) error {
	var (
		network = &Network{}
		schema  = c.GlobalString("schema")
		err     error
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	outfile, err := os.Create(schema + ".dot")
	if err != nil {
		return err
	}

	return networkGraph(*network, outfile)
}

func cmdCreateNetwork(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	ipRange := c.String("cidr")
	if ipRange == "" {
		return fmt.Errorf("ip range cannot be empty")
	}

	ipnet, err := types.ParseIPNet(ipRange)
	if err != nil {
		errors.Wrap(err, "invalid ip range")
	}
	network := pkg.Network{
		Name:         name,
		IPRange:      ipnet,
		NetResources: []pkg.NetResource{},
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("schema"), r)
}

func cmdsAddNode(c *cli.Context) error {
	var (
		network = &Network{}
		schema  = c.GlobalString("schema")
		err     error

		nodeID = c.String("node")
		subnet = c.String("subnet")
		port   = c.Uint("port")

		forceHidden = c.Bool("force-hidden")
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	if subnet == "" {
		return fmt.Errorf("subnet cannot be empty")
	}
	ipnet, err := types.ParseIPNet(subnet)
	if err != nil {
		return errors.Wrap(err, "invalid subnet")
	}

	if port == 0 {
		port, err = pickPort(nodeID)
		if err != nil {
			return errors.Wrap(err, "failed to pick wireguard port")
		}
	}

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return errors.Wrap(err, "error during wireguard key generation")
	}
	sk := privateKey.String()

	pk, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to parse nodeID")
	}

	encrypted, err := crypto.Encrypt([]byte(sk), pk)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt private key")
	}

	pubSubnets, err := getEndPointAddrs(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to get node public endpoints")
	}
	var endpoints []net.IP
	if !forceHidden {
		for _, sn := range pubSubnets {
			endpoints = append(endpoints, sn.IP)
		}
	}

	nr := NetResource{
		NetResource: pkg.NetResource{
			NodeID:       nodeID,
			Subnet:       ipnet,
			WGListenPort: uint16(port),
			WGPublicKey:  privateKey.PublicKey().String(),
			WGPrivateKey: hex.EncodeToString(encrypted),
		},
		PubEndpoints: endpoints,
	}

	network.NetResources = append(network.NetResources, nr)

	if err = generatePeers(network); err != nil {
		return errors.Wrap(err, "failed to generate peers")
	}

	r, err := embed(pkgNetFromNetwork(*network), provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
}

func cmdsAddAccess(c *cli.Context) error {
	var (
		network = &Network{}
		schema  = c.GlobalString("schema")
		err     error

		nodeID   = c.String("node")
		subnet   = c.String("subnet")
		wgPubKey = c.String("wgpubkey")

		ip4 = c.Bool("ip4")
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	if subnet == "" {
		return fmt.Errorf("subnet cannot be empty")
	}
	ipnet, err := types.ParseIPNet(subnet)
	if err != nil {
		return errors.Wrap(err, "invalid subnet")
	}

	var nodeExists bool
	var node NetResource
	for _, nr := range network.NetResources {
		if nr.NodeID == nodeID {
			node = nr
			nodeExists = true
			break
		}
	}

	if !nodeExists {
		return errors.New("can not add access through a node which is not in the network")
	}

	if len(node.PubEndpoints) == 0 {
		return errors.New("access node must have at least 1 public endpoint")
	}

	var endpoint string
	for _, ep := range node.PubEndpoints {
		if ep.To4() != nil {
			// ipv4 address
			if ip4 {
				endpoint = fmt.Sprintf("%s:%d", ep.String(), node.WGListenPort)
				break
			}
			// we want ipv6 so use the next address
			continue
		}
		if ep.To16() != nil {
			// due to the previous branch this can now only be an ipv6 address
			if !ip4 {
				endpoint = fmt.Sprintf("[%s]:%d", node.PubEndpoints[0].String(), node.WGListenPort)
				break
			}
			// we want ipv4 so use next address
			continue
		}
	}
	if endpoint == "" {
		return errors.New("access node has no public endpoint of the requested type")
	}

	var privateKey wgtypes.Key
	if wgPubKey == "" {
		privateKey, err = wgtypes.GeneratePrivateKey()
		if err != nil {
			return errors.Wrap(err, "error during wireguard key generation")
		}
		wgPubKey = privateKey.PublicKey().String()
	}

	ap := AccessPoint{
		NodeID:      nodeID,
		Subnet:      ipnet,
		WGPublicKey: wgPubKey,
		IP4:         ip4,
	}

	network.AccessPoints = append(network.AccessPoints, ap)

	if err = generatePeers(network); err != nil {
		return errors.Wrap(err, "failed to generate peers")
	}

	wgConf, err := genWGQuick(privateKey.String(), ipnet, node.WGPublicKey, network.IPRange, endpoint)
	if err != nil {
		return err
	}

	fmt.Println(wgConf)

	r, err := embed(pkgNetFromNetwork(*network), provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
}

func cmdsRemoveNode(c *cli.Context) error {
	var (
		network = &Network{}
		schema  = c.GlobalString("schema")
		nodeID  = c.String("node")
		err     error
	)

	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	for i, nr := range network.NetResources {
		if nr.NodeID == nodeID {
			network.NetResources = append(network.NetResources[:i], network.NetResources[i+1:]...)
			break
		}
	}

	// we don't remove the peer from the other network resource
	// while this is dirty wireguard doesn't really care
	raw, err := json.Marshal(pkgNetFromNetwork(*network))
	if err != nil {
		return err
	}

	r := &provision.Reservation{
		Type: provision.NetworkReservation,
		Data: raw,
	}
	// r, err := embed(network, provision.NetworkReservation)
	// if err != nil {
	//	return err
	// }

	return output(schema, r)
}

func loadNetwork(name string) (*Network, error) {
	network := &pkg.Network{}

	if name == "" {
		return nil, fmt.Errorf("schema name cannot be empty")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := &provision.Reservation{}
	if err := json.NewDecoder(f).Decode(r); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded reservation at %s", name)
	}

	if err := json.Unmarshal(r.Data, network); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded network at %s", name)
	}

	net := networkFromPkgNet(*network)

	if err = setPubEndpoints(&net); err != nil {
		return nil, err
	}

	extractAccessPoints(&net)

	return &net, nil
}

func pickPort(nodeID string) (uint, error) {
	node, err := client.GetNode(pkg.StrIdentifier(nodeID))
	if err != nil {
		return 0, err
	}

	p := uint(rand.Intn(6000) + 2000)

	for isIn(node.WGPorts, p) {
		p = uint(rand.Intn(6000) + 2000)
	}
	return p, nil
}

// a node has either a public namespace with []ipv4 or/and []ipv6 -or-
// some interface has received a SLAAC addr
// which has been registered in BCDB
func getEndPointAddrs(nodeID pkg.Identifier) ([]types.IPNet, error) {
	node, err := client.GetNode(nodeID)
	if err != nil {
		return nil, err
	}
	var endpoints []types.IPNet
	if node.PublicConfig != nil {
		if node.PublicConfig.IPv4.IP != nil {
			ip := node.PublicConfig.IPv4.IP
			if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
				endpoints = append(endpoints, node.PublicConfig.IPv4)
			}
		}
		if node.PublicConfig.IPv6.IP != nil {
			ip := node.PublicConfig.IPv6.IP
			if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
				endpoints = append(endpoints, node.PublicConfig.IPv6)
			}
		}
	} else {
		for _, iface := range node.Ifaces {
			for _, ip := range iface.Addrs {
				if !ip.IP.IsGlobalUnicast() || isPrivateIP(ip.IP) {
					continue
				}
				endpoints = append(endpoints, ip)
			}
		}
	}
	// If the length is 0, then its a hidden node
	return endpoints, nil
}

func isIn(l []uint, i uint) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func hasIPv4(n NetResource) bool {
	for _, pep := range n.PubEndpoints {
		if pep.To4() != nil {
			return true
		}
	}
	return false
}

// This function assumes:
// - that a hidden node has functioning IPv4
// - that a public node ALWAYS has public IPv6, and OPTIONALLY public IPv4
// - that any public endpoint on any node is actually reachable (i.e. no firewall
//		blocking incoming traffic)
func generatePeers(n *Network) error {
	// Find public node, which will be used to connect all hidden nodes.
	// In case there are hidden nodes, the public node needs IPv4 support as well.
	var hasHiddenNodes bool
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) == 0 {
			hasHiddenNodes = true
			break
		}
	}

	// Look for a public node to connect hidden nodes. This is only needed
	// in case there are hidden nodes.
	var pubNr string
	if hasHiddenNodes {
		for _, nr := range n.NetResources {
			if hasIPv4(nr) {
				pubNr = nr.NodeID
				break
			}
		}
		if pubNr == "" {
			return errors.New("Network has hidden nodes but no public IPv4 node exists")
		}
	}

	// We also need to inform nodes how to route the external access subnets.
	// Working with the knowledge that these external subnets come in through
	// the network through a single access point, which is part of the network
	// and thus already routed, we can map the external subnets to the subnet
	// of the access point, and add these external subnets to all peers who also
	// have the associated internal subnet.
	//
	// Map the network subnets to their respective node ids first for easy access later
	internalSubnets := make(map[string]types.IPNet)
	for _, nr := range n.NetResources {
		internalSubnets[nr.NodeID] = nr.Subnet
	}

	externalSubnets := make(map[string][]types.IPNet) // go does not like `types.IPNet` as key
	for _, ap := range n.AccessPoints {
		externalSubnets[internalSubnets[ap.NodeID].String()] = append(externalSubnets[internalSubnets[ap.NodeID].String()], ap.Subnet)
	}

	// Maintain a mapping of access point nodes to the subnet and wg key they give access
	// to, as these need to be added as peers as well for these nodes
	accessPoints := make(map[string][]AccessPoint)
	for _, ap := range n.AccessPoints {
		accessPoints[ap.NodeID] = append(accessPoints[ap.NodeID], ap)
	}

	// Find all hidden nodes, and collect their subnets. Also collect the subnets
	// of public IPv6 only nodes, since hidden nodes need IPv4 to connect.
	hiddenSubnets := make(map[string]types.IPNet)
	// also maintain subnets from nodes who have only IPv6 since this will also
	// need to be routed for hidden nodes
	ipv6OnlySubnets := make(map[string]types.IPNet)
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) == 0 {
			hiddenSubnets[nr.NodeID] = nr.Subnet
			continue
		}
		if !hasIPv4(nr) {
			ipv6OnlySubnets[nr.NodeID] = nr.Subnet
		}
	}

	for i := range n.NetResources {
		// Note: we need to loop by index and manually assign nr, doing
		// for _, nr := range ... causes nr to be copied, meaning we can't modify
		// it in place
		nr := &n.NetResources[i]
		nr.Peers = []pkg.Peer{}
		for _, onr := range n.NetResources {
			if nr.NodeID == onr.NodeID {
				continue
			}

			allowedIPs := make([]types.IPNet, 2)
			allowedIPs[0] = onr.Subnet
			allowedIPs[1] = types.NewIPNet(wgIP(&onr.Subnet.IPNet))

			var endpoint string

			if len(nr.PubEndpoints) == 0 {
				// If node is hidden, set only public peers (with IPv4), and set first public peer to
				// contain all hidden subnets, except for the one owned by the node
				if !hasIPv4(onr) {
					continue
				}

				// Also add all other subnets if this is the pub node
				if onr.NodeID == pubNr {
					for owner, subnet := range hiddenSubnets {
						// Do not add our own subnet
						if owner == nr.NodeID {
							continue
						}

						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}

					for _, subnet := range ipv6OnlySubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}
				}

				// Endpoint must be IPv4
				for _, pep := range onr.PubEndpoints {
					if pep.To4() != nil {
						endpoint = fmt.Sprintf("%s:%d", pep.String(), onr.WGListenPort)
						break
					}
				}
			} else if len(onr.PubEndpoints) == 0 && hasIPv4(*nr) {
				// if the peer is hidden but we have IPv4,  we can connect to it, but we don't know
				// an endpoint.
				endpoint = ""
			} else {
				// if we are not hidden, we add all other nodes, unless we don't
				// have IPv4, because then we also can't connect to hidden nodes.
				// Ignore hidden nodes if we don't have IPv4
				if !hasIPv4(*nr) && len(onr.PubEndpoints) == 0 {
					continue
				}

				// both nodes are public therefore we can connect over IPv6

				// if this is the selected pubNr - also need to add allowedIPs
				// for the hidden nodes
				if onr.NodeID == pubNr {
					for _, subnet := range hiddenSubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}
				}

				// Since the node is not hidden, we know that it MUST have at least
				// 1 IPv6 address
				for _, pep := range onr.PubEndpoints {
					if pep.To4() == nil && pep.To16() != nil {
						endpoint = fmt.Sprintf("[%s]:%d", pep.String(), onr.WGListenPort)
						break
					}
				}

				// as a fallback assign IPv4
				if endpoint == "" {
					for _, pep := range onr.PubEndpoints {
						if pep.To4() != nil {
							endpoint = fmt.Sprintf("%s:%d", pep.String(), onr.WGListenPort)
							break
						}
					}
				}
			}

			// Add subnets for external access
			for i := 0; i < len(allowedIPs); i++ {
				for _, subnet := range externalSubnets[allowedIPs[i].String()] {
					allowedIPs = append(allowedIPs, subnet)
					allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
				}
			}

			nr.Peers = append(nr.Peers, pkg.Peer{
				WGPublicKey: onr.WGPublicKey,
				Subnet:      onr.Subnet,
				AllowedIPs:  allowedIPs,
				Endpoint:    endpoint,
			})
		}

		// Add configured external access peers
		for _, ea := range accessPoints[nr.NodeID] {
			allowedIPs := make([]types.IPNet, 2)
			allowedIPs[0] = ea.Subnet
			allowedIPs[1] = types.NewIPNet(wgIP(&ea.Subnet.IPNet))

			nr.Peers = append(nr.Peers, pkg.Peer{
				WGPublicKey: ea.WGPublicKey,
				Subnet:      ea.Subnet,
				AllowedIPs:  allowedIPs,
				Endpoint:    "",
			})
		}
	}

	return nil
}

func isIPv4Subnet(n types.IPNet) bool {
	ones, bits := n.IPNet.Mask.Size()
	if bits != 32 {
		return false
	}
	return ones <= 30
}

func genWGQuick(wgPrivateKey string, localSubnet types.IPNet, peerWgPubKey string, allowedSubnet types.IPNet, peerEndpoint string) (string, error) {
	type data struct {
		PrivateKey    string
		Address       string
		PeerWgPubKey  string
		AllowedSubnet string
		PeerEndpoint  string
	}

	if !isIPv4Subnet(localSubnet) {
		return "", errors.New("local subnet is not a valid IPv4 subnet")
	}

	tmpl, err := template.New("wg").Parse(wgTmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}

	if err := tmpl.Execute(buf, data{
		PrivateKey:    wgPrivateKey,
		Address:       types.NewIPNet(wgIP(&localSubnet.IPNet)).String(),
		PeerWgPubKey:  peerWgPubKey,
		AllowedSubnet: strings.Join([]string{allowedSubnet.String(), types.NewIPNet(wgSubnet(&allowedSubnet.IPNet)).String()}, ","),
		PeerEndpoint:  peerEndpoint,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var wgTmpl = `
[Interface]
PrivateKey = {{.PrivateKey}}
Address = {{.Address}}

[Peer]
PublicKey = {{.PeerWgPubKey}}
AllowedIPs = {{.AllowedSubnet}}
PersistentKeepalive = 20
{{if .PeerEndpoint}}Endpoint = {{.PeerEndpoint}}{{end}}
`

func networkGraph(n Network, w io.Writer) error {
	nodes := make(map[string]dot.Node)
	nodesByID := make(map[string]dot.Node)
	graph := dot.NewGraph(dot.Directed)

	for _, nr := range n.NetResources {
		node := graph.Node(strings.Join([]string{nr.NodeID, nr.Subnet.String()}, "\n")).Box()
		// set special style for "hidden" nodes
		if len(nr.PubEndpoints) == 0 {
			node.Attr("style", "dashed")
			node.Attr("color", "blue")
			graph.AddToSameRank("hidden nodes", node)
		}
		nodes[nr.WGPublicKey] = node
		nodesByID[nr.NodeID] = node
	}

	// add external access
	for _, ea := range n.AccessPoints {
		node := graph.Node(strings.Join([]string{"External network", ea.Subnet.String()}, "\n")).Box()
		// set style for hidden nodes
		node.Attr("style", "dashed")
		node.Attr("color", "green")
		graph.AddToSameRank("external access", node)
		// add link to access point
		edge := graph.Edge(node, nodesByID[ea.NodeID], n.IPRange.String())
		if ea.IP4 {
			edge.Attr("color", "blue")
		}
		nodes[ea.WGPublicKey] = node
	}

	for _, nr := range n.NetResources {
		for _, peer := range nr.Peers {
			allowedIPs := make([]string, 0, len(peer.AllowedIPs)/2)
			for _, aip := range peer.AllowedIPs {
				if !isCGN(aip) {
					allowedIPs = append(allowedIPs, aip.String())
				}
			}

			edge := graph.Edge(nodes[nr.WGPublicKey], nodes[peer.WGPublicKey], strings.Join(allowedIPs, "\n"))
			if peer.Endpoint == "" {
				// connections to this peer are IPv4 -> blue, and can not be initiated by this node -> dashed
				edge.Attr("color", "blue").Attr("style", "dashed")
				continue
			}
			if net.ParseIP(peer.Endpoint[:strings.LastIndex(peer.Endpoint, ":")]).To4() != nil {
				// IPv4 connection -> blue
				edge.Attr("color", "blue")
			}
		}
	}

	graph.Write(w)
	return nil
}

func wgIP(subnet *net.IPNet) *net.IPNet {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	return &net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(32, 32),
	}
}

func wgSubnet(subnet *net.IPNet) *net.IPNet {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	ones, _ := subnet.Mask.Size()

	return &net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(ones+8, 32),
	}
}

func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{}
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func isCGN(subnet types.IPNet) bool {
	_, block, err := net.ParseCIDR("100.64.0.0/10")
	if err != nil {
		panic(err)
	}
	return block.Contains(subnet.IP)
}

func setPubEndpoints(n *Network) error {
	for i := range n.NetResources {
		pep, err := getEndPointAddrs(pkg.StrIdentifier(n.NetResources[i].NodeID))
		if err != nil {
			return err
		}

		var endpoints []net.IP
		for _, sn := range pep {
			endpoints = append(endpoints, sn.IP)
		}

		n.NetResources[i].PubEndpoints = endpoints
	}

	// remove the pub endpoints from nodes which we assume have been marked
	// as force hidden
	hiddenNodes := make(map[string]struct{})
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) > 0 {
			for _, peer := range nr.Peers {
				if peer.Endpoint == "" {
					hiddenNodes[peer.WGPublicKey] = struct{}{}
				}
			}
		}
	}

	for i := range n.NetResources {
		if _, exists := hiddenNodes[n.NetResources[i].WGPublicKey]; exists {
			n.NetResources[i].PubEndpoints = nil
		}
	}

	return nil
}

func extractAccessPoints(n *Network) {
	// gather all actual nodes, using their wg pubkey as key in the map (NodeID
	// can't be seen in the actual peer struct)
	actualNodes := make(map[string]struct{})
	for _, nr := range n.NetResources {
		actualNodes[nr.WGPublicKey] = struct{}{}
	}

	aps := []AccessPoint{}
	for _, nr := range n.NetResources {
		for _, peer := range nr.Peers {
			if _, exists := actualNodes[peer.WGPublicKey]; !exists {
				// peer is not a node so it must be external
				aps = append(aps, AccessPoint{
					NodeID:      nr.NodeID,
					Subnet:      peer.Subnet,
					WGPublicKey: peer.WGPublicKey,
					// we can't infer if we use IPv6 or IPv4
				})
			}
		}
	}
	n.AccessPoints = aps
}
