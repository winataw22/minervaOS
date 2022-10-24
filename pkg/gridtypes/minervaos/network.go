package zos

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// NetID is a type defining the ID of a network
type NetID string

func (i NetID) String() string {
	return string(i)
}

// NetworkID construct a network ID based on a userID and network name
func NetworkID(twin uint32, network gridtypes.Name) NetID {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprint(twin))
	buf.WriteString(":")
	buf.WriteString(string(network))
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return NetID(string(b))
}

func NetworkIDFromWorkloadID(wl gridtypes.WorkloadID) (NetID, error) {
	twin, _, name, err := wl.Parts()
	if err != nil {
		return "", err
	}
	return NetworkID(twin, name), nil
}

// Network is the description of a part of a network local to a specific node.
// A network workload defines a wireguard network that is usually spans multiple nodes. One of the nodes must work as an access node
// in other words, it must be reachable from other nodes, hence it needs to have a `PublicConfig`.
// Since the user library creates all deployments upfront then all wireguard keys, and ports must be pre-determinstic and must be
// also created upfront.
// A network structure basically must consist of
// - The network information (IP range) must be an ipv4 /16 range
// - The local (node) peer definition (subnet of the network ip range, wireguard secure key, wireguard port if any)
// - List of other peers that are part of the same network with their own config
// - For each PC or a laptop (for each wireguard peer) there must be a peer in the peer list (on all nodes)
// This is why this can get complicated.
type Network struct {
	// IP range of the network, must be an IPv4 /16
	// for example a 10.1.0.0/16
	NetworkIPRange gridtypes.IPNet `json:"ip_range"`

	// IPV4 subnet for this network resource
	// this must be a valid subnet of the entire network ip range.
	// for example 10.1.1.0/24
	Subnet gridtypes.IPNet `json:"subnet"`

	// The private wg key of this node (this peer) which is installing this
	// network workload right now.
	// This has to be filled in by the user (and not generated for example)
	// because other peers need to be installed as well (with this peer public key)
	// hence it's easier to configure everything one time at the user side and then
	// apply everything on all nodes at once
	WGPrivateKey string `json:"wireguard_private_key"`
	// WGListenPort is the wireguard listen port on this node. this has
	// to be filled in by the user for same reason as private key (other nodes need to know about it)
	// To find a free port you have to ask the node first by a call over RMB about which ports are possible
	// to use.
	WGListenPort uint16 `json:"wireguard_listen_port"`

	// Peers is a list of other peers in this network
	Peers []Peer `json:"peers"`
}

// Valid checks if the network resource is valid.
func (n Network) Valid(getter gridtypes.WorkloadGetter) error {

	if n.NetworkIPRange.Nil() {
		return fmt.Errorf("network IP range cannot be empty")
	}

	if len(n.Subnet.IP) == 0 {
		return fmt.Errorf("network resource subnet cannot empty")
	}

	if n.WGPrivateKey == "" {
		return fmt.Errorf("network resource wireguard private key cannot empty")
	}

	for _, peer := range n.Peers {
		if err := peer.Valid(); err != nil {
			return err
		}
	}

	return nil
}

// Challenge implements WorkloadData
func (n Network) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", n.NetworkIPRange.String()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", n.Subnet.String()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", n.WGPrivateKey); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%d", n.WGListenPort); err != nil {
		return err
	}

	for _, p := range n.Peers {
		if err := p.Challenge(b); err != nil {
			return err
		}
	}

	return nil
}

// Capacity implementation
func (n Network) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{}, nil
}

// Peer is the description of a peer of a NetResource
type Peer struct {
	// IPV4 subnet of the network resource of the peer
	Subnet gridtypes.IPNet `json:"subnet"`
	// WGPublicKey of the peer (driven from its private key)
	WGPublicKey string `json:"wireguard_public_key"`
	// Allowed Ips is related to his subnet.
	// todo: remove and derive from subnet
	AllowedIPs []gridtypes.IPNet `json:"allowed_ips"`
	// Entrypoint of the peer
	Endpoint string `json:"endpoint"`
}

// Valid checks if peer is valid
func (p *Peer) Valid() error {
	if p.Subnet.Nil() {
		return fmt.Errorf("peer wireguard subnet cannot empty")
	}

	if len(p.AllowedIPs) <= 0 {
		return fmt.Errorf("peer wireguard allowedIPs cannot empty")
	}

	if p.WGPublicKey == "" {
		return fmt.Errorf("peer wireguard public key cannot empty")
	}

	return nil
}

// Challenge for peer
func (p Peer) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", p.WGPublicKey); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", p.Endpoint); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", p.Subnet.String()); err != nil {
		return err
	}
	for _, ip := range p.AllowedIPs {
		if _, err := fmt.Fprintf(w, "%s", ip.String()); err != nil {
			return err
		}
	}
	return nil
}
