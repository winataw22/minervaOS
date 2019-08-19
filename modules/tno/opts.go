package tno

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/dspinhirne/netaddr-go"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/crypto"
	"github.com/threefoldtech/zosv2/modules/identity"
)

// Opts is a function applies on a network object that configure
// a part of the network
type Opts func(n *modules.Network) error

// Configure is the method used to apply a list of Opts on a network object
func Configure(n *modules.Network, opts []Opts) error {
	for _, opt := range opts {
		if err := opt(n); err != nil {
			return err
		}
	}
	return nil
}

// GenerateID generates a new random NetID and set it on the network object
// if the network object already have a NetID set, an error is returned
func GenerateID() Opts {
	return func(n *modules.Network) error {
		if n.NetID != "" {
			return fmt.Errorf("network already has a NetID set")
		}
		k, err := identity.GenerateKeyPair()

		if err != nil {
			return err
		}
		n.NetID = modules.NetID(k.Identity())
		return nil
	}
}

// ConfigurePrefixZero sets the PrefixZero field of the TNO
// farm allocation is the allocation the farmer has given for his farm
func ConfigurePrefixZero(farmAllocation *net.IPNet) Opts {
	return func(n *modules.Network) error {

		ipv6net, err := netaddr.ParseIPv6Net(farmAllocation.String())
		if err != nil {
			return err
		}
		subnet := ipv6net.NthSubnet(64, 0)

		_, zeroNet, err := net.ParseCIDR(subnet.String())
		if err != nil {
			return err
		}

		n.PrefixZero = zeroNet
		return nil
	}
}

// ConfigureExitResource configure the exit point of the TNO
// nodeID is the ID of the node used as exit point
// allocation if the /64 allocation for the exit resource
// key is the wireguard key pair for the wiregard exit peer
func ConfigureExitResource(nodeID string, allocation *net.IPNet, publicIP net.IP, key wgtypes.Key, farmAllocSize int) Opts {
	return func(n *modules.Network) error {
		if n.PrefixZero == nil {
			return fmt.Errorf("cannot add a node when the network object does not have a PrefixZero set")
		}

		exitNibble := newNibble(allocation, farmAllocSize)

		pk, err := crypto.KeyFromID(modules.StrIdentifier(nodeID))
		if err != nil {
			return errors.Wrapf(err, "failed to extract public key from node ID %s", nodeID)
		}
		privateKey, err := crypto.Encrypt([]byte(key.String()), pk)
		if err != nil {
			return err
		}

		exitPeer := &modules.Peer{
			Type:   modules.ConnTypeWireguard,
			Prefix: allocation,
			Connection: modules.Wireguard{
				IP:         publicIP,
				Port:       exitNibble.WireguardPort(),
				Key:        key.PublicKey().String(),
				PrivateKey: fmt.Sprintf("%x", privateKey),
			},
		}

		n.Resources = append(n.Resources, &modules.NetResource{
			NodeID: &modules.NodeID{
				ID: nodeID,
				// FarmerID:       networkReq.ExitFarm,
				ReachabilityV6: modules.ReachabilityV6Public,
				ReachabilityV4: modules.ReachabilityV4Public,
			},
			Prefix: allocation,
			LinkLocal: &net.IPNet{
				IP:   exitNibble.fe80(),
				Mask: net.CIDRMask(64, 128),
			},
			Peers:     []*modules.Peer{exitPeer},
			ExitPoint: true,
		})

		n.Exit = &modules.ExitPoint{
			Ipv6Conf: &modules.Ipv6Conf{
				Addr: &net.IPNet{
					IP:   exitNibble.fe80(),
					Mask: net.CIDRMask(64, 128),
				},
				Gateway: net.ParseIP("fe80::1"),
				Iface:   "public",
			},
		}
		return nil
	}
}

// AddNode adds a network resource to the TNO
// if the node is a publicly accessible node, publicIP and port needs to be not nil
func AddNode(nodeID string, farmID string, allocation *net.IPNet, key wgtypes.Key, publicIP net.IP) Opts {
	return func(n *modules.Network) error {

		if n.PrefixZero == nil {
			return fmt.Errorf("cannot add a node when the network object does not have a PrefixZero set")
		}

		allocSize, _ := n.PrefixZero.Mask.Size()
		exitNibble := newNibble(allocation, allocSize)

		v6Reach := modules.ReachabilityV6ULA
		if publicIP != nil {
			v6Reach = modules.ReachabilityV6Public
		}

		var peers []*modules.Peer
		if len(n.Resources) > 0 {
			peers = n.Resources[0].Peers
		}

		pk, err := crypto.KeyFromID(modules.StrIdentifier(nodeID))
		if err != nil {
			return errors.Wrapf(err, "failed to extract public key from node ID %s", nodeID)
		}
		privateKey, err := crypto.Encrypt([]byte(key.String()), pk)
		if err != nil {
			return err
		}

		resource := &modules.NetResource{
			NodeID: &modules.NodeID{
				ID:             nodeID,
				FarmerID:       farmID,
				ReachabilityV6: v6Reach,
				ReachabilityV4: modules.ReachabilityV4Hidden, //TODO change once we support ipv4 public nodes
			},
			Prefix: allocation,
			LinkLocal: &net.IPNet{
				IP:   exitNibble.fe80(),
				Mask: net.CIDRMask(64, 128),
			},
			Peers:     peers,
			ExitPoint: false,
		}

		peer := &modules.Peer{
			Type:   modules.ConnTypeWireguard,
			Prefix: allocation,
			Connection: modules.Wireguard{
				Port:       exitNibble.WireguardPort(),
				Key:        key.PublicKey().String(),
				PrivateKey: fmt.Sprintf("%x", privateKey),
			},
		}
		if publicIP != nil {
			peer.Connection.IP = publicIP
		}

		n.Resources = append(n.Resources, resource)
		for _, r := range n.Resources {
			r.Peers = append(r.Peers, peer)
		}

		return nil
	}

}

// AddUser adds a new member that is not a 0-OS node to the TNO
// This is mainly use to allow client to connect into their private network
func AddUser(userID string, allocation *net.IPNet, key wgtypes.Key) Opts {
	return func(n *modules.Network) error {

		if n.PrefixZero == nil {
			return fmt.Errorf("cannot add a node when the network object does not have a PrefixZero set")
		}

		allocSize, _ := n.PrefixZero.Mask.Size()
		exitNibble := newNibble(allocation, allocSize)

		var peers []*modules.Peer
		if len(n.Resources) > 0 {
			peers = n.Resources[0].Peers
		}

		pk, err := crypto.KeyFromID(modules.StrIdentifier(userID))
		if err != nil {
			return errors.Wrapf(err, "failed to extract public key from user ID %s", userID)
		}
		privateKey, err := crypto.Encrypt([]byte(key.String()), pk)
		if err != nil {
			return err
		}

		resource := &modules.NetResource{
			NodeID: &modules.NodeID{
				ID: userID,
				// FarmerID:       exitFarm,
				ReachabilityV6: modules.ReachabilityV6ULA,
				ReachabilityV4: modules.ReachabilityV4Hidden,
			},
			Prefix: allocation,
			LinkLocal: &net.IPNet{
				IP:   exitNibble.fe80(),
				Mask: net.CIDRMask(64, 128),
			},
			Peers:     peers,
			ExitPoint: false,
		}

		peer := &modules.Peer{
			Type:   modules.ConnTypeWireguard,
			Prefix: allocation,
			Connection: modules.Wireguard{
				Key:        key.PublicKey().String(),
				PrivateKey: fmt.Sprintf("%x", privateKey),
			},
		}

		n.Resources = append(n.Resources, resource)
		for _, r := range n.Resources {
			r.Peers = append(r.Peers, peer)
		}

		return nil
	}
}

// RemoveNode remove a network resource associated with nodeID and
// all the occurrence of this peer in all other resources
func RemoveNode(nodeID string) Opts {

	return func(n *modules.Network) error {
		var prefix *net.IPNet
		for i, r := range n.Resources {
			if r.NodeID.ID == nodeID {
				prefix = r.Prefix
				n.Resources = removeResource(n.Resources, i)
			}
		}

		if prefix == nil {
			// node ID not present in the TNO
			return nil
		}

		for _, r := range n.Resources {
			for i, peer := range r.Peers {
				if peer.Prefix.String() == prefix.String() {
					r.Peers = removePeer(r.Peers, i)
					break
				}
			}

		}

		return nil
	}
}

func removeResource(s []*modules.NetResource, i int) []*modules.NetResource {
	return append(s[:i], s[i+1:]...)
}

func removePeer(s []*modules.Peer, i int) []*modules.Peer {
	return append(s[:i], s[i+1:]...)
}
