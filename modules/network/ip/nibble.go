package ip

import (
	"fmt"
	"net"
)

// Nibble is an helper struct used to generate
// deterministic name based on an IPv6 address
type Nibble struct {
	allocNr int8
	nibble  []byte
	// hexNibble string
}

// NewNibble create a new Nibble object
func NewNibble(prefix *net.IPNet, allocNr int8) *Nibble {

	var b []byte
	size, _ := prefix.Mask.Size()
	if size < 48 {
		b = []byte(prefix.IP)[4:8]
	} else {
		b = []byte(prefix.IP)[6:8]
	}

	return &Nibble{
		nibble:  b,
		allocNr: allocNr,
	}
}

// Hex return the hexadecimal version of the meaningfull nibble
func (n *Nibble) Hex() string {
	if len(n.nibble) == 2 {
		return fmt.Sprintf("%x", n.nibble)
	}
	if len(n.nibble) == 4 {
		return fmt.Sprintf("%x:%x", n.nibble[:2], n.nibble[2:])
	}
	panic("wrong nibble size")
}

func (n *Nibble) nocolonhex() string {
	if len(n.nibble) == 2 {
		return fmt.Sprintf("%x", n.nibble)
	}
	if len(n.nibble) == 4 {
		return fmt.Sprintf("%x%x", n.nibble[:2], n.nibble[2:])
	}
	panic("wrong nibble size")
}

// WiregardName return the deterministic wireguard name
func (n *Nibble) WiregardName() string {
	return fmt.Sprintf("wg-%s-%d", n.nocolonhex(), n.allocNr)
}

// BridgeName return the deterministic bridge name
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.nocolonhex(), n.allocNr)
}

// NetworkName return the deterministic network name
func (n *Nibble) NetworkName() string {
	return fmt.Sprintf("net-%s-%d", n.nocolonhex(), n.allocNr)
}

// VethName return the deterministic veth interface name
func (n *Nibble) VethName() string {
	return fmt.Sprintf("veth-%s-%d", n.nocolonhex(), n.allocNr)
}

// PubName return the deterministic public interface name
func (n *Nibble) PubName() string {
	return fmt.Sprintf("pub-%s-%d", n.nocolonhex(), n.allocNr)
}

// ToV4 returns 2 uint8 that can be used as the last 2 bytes of an IPv4 address
func (n *Nibble) ToV4() (uint8, uint8, error) {
	if len(n.nibble) > 2 {
		return 0, 0, fmt.Errorf("cannot call ToV4 with a nibble of size %d", len(n.nibble))
	}
	return n.nibble[0], n.nibble[1], nil
}
