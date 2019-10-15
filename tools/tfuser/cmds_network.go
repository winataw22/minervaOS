package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/provision"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
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
	network := &pkg.Network{
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
		network = &pkg.Network{}
		schema  = c.GlobalString("schema")
		err     error

		nodeID = c.String("node")
		subnet = c.String("subnet")
		port   = c.Uint("port")
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
	fmt.Println("subnet", subnet)
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

	nr := pkg.NetResource{
		NodeID:       nodeID,
		Subnet:       ipnet,
		WGListenPort: uint16(port),
		WGPublicKey:  privateKey.PublicKey().String(),
		WGPrivateKey: hex.EncodeToString(encrypted),
	}

	network.NetResources = append(network.NetResources, nr)

	for i, nr := range network.NetResources {
		network.NetResources[i].Peers = generatePeers(nr.NodeID, *network)
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
}

func cmdsRemoveNode(c *cli.Context) error {
	var (
		network = &pkg.Network{}
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
	raw, err := json.Marshal(network)
	if err != nil {
		return err
	}

	r := &provision.Reservation{
		Type: provision.NetworkReservation,
		Data: raw,
	}
	// r, err := embed(network, provision.NetworkReservation)
	// if err != nil {
	// 	return err
	// }

	return output(schema, r)
}

func loadNetwork(name string) (network *pkg.Network, err error) {
	network = &pkg.Network{}

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
	return network, nil
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

func isIn(l []uint, i uint) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func generatePeers(nodeID string, n pkg.Network) []pkg.Peer {
	peers := make([]pkg.Peer, 0, len(n.NetResources))
	for _, nr := range n.NetResources {
		if nr.NodeID == nodeID {
			continue
		}

		allowedIPs := make([]types.IPNet, 2)
		allowedIPs[0] = nr.Subnet
		allowedIPs[1] = types.NewIPNet(wgIP(&nr.Subnet.IPNet))

		peers = append(peers, pkg.Peer{
			WGPublicKey: nr.WGPublicKey,
			AllowedIPs:  allowedIPs,
		})
	}
	return peers
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
