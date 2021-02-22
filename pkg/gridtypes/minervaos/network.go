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
func NetworkID(user, network string) NetID {
	buf := bytes.Buffer{}
	buf.WriteString(user)
	buf.WriteString(":")
	buf.WriteString(network)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return NetID(string(b))
}

// Network is the description of a part of a network local to a specific node
type Network struct {
	Name string `json:"name"`
	// IP range of the network, must be an IPv4 /16
	NetworkIPRange gridtypes.IPNet `json:"ip_range"`

	// IPV4 subnet for this network resource
	Subnet gridtypes.IPNet `json:"subnet"`

	WGPrivateKeyEncrypted string `json:"wireguard_private_key_encrypted"`
	// WGPublicKey           string `json:"wireguard_public_key"`
	WGListenPort uint16 `json:"wireguard_listen_port"`

	Peers []Peer `json:"peers"`
}

// Valid checks if the network resource is valid.
func (n Network) Valid() error {

	if n.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	if n.NetworkIPRange.Nil() {
		return fmt.Errorf("network IP range cannot be empty")
	}

	if len(n.Subnet.IP) == 0 {
		return fmt.Errorf("network resource subnet cannot empty")
	}

	if n.WGPrivateKeyEncrypted == "" {
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
	if _, err := fmt.Fprintf(b, "%s", n.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", n.NetworkIPRange.String()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", n.Subnet.String()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", n.WGPrivateKeyEncrypted); err != nil {
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

	WGPublicKey string            `json:"wireguard_public_key"`
	AllowedIPs  []gridtypes.IPNet `json:"allowed_ips"`
	Endpoint    string            `json:"endpoint"`
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

//Challenge for peer
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
