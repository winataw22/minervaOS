package tno

import (
	"encoding/binary"
	"fmt"
	"net"
)

type nibble []byte

func (n nibble) Hex() string {
	if len(n) == 2 {
		return fmt.Sprintf("%x", n)
	}
	if len(n) == 4 {
		return fmt.Sprintf("%x:%x", n[:2], n[2:])
	}
	panic("wrong nibble size")
}

func (n nibble) fe80() net.IP {
	return net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex()))
}

// IP return the IP passed as argument with the
// last bits replaced with the value of nibble
func (n nibble) IP(ip net.IP) net.IP {
	i := make([]byte, net.IPv6len)
	copy(i[:], ip)
	copy(i[net.IPv6len-len(n):], n)
	return net.IP(i[:])
}

// WireguardPort return the deterministic wireguard listen port
func (n nibble) WireguardPort() uint16 {
	return binary.BigEndian.Uint16(n)
}

func newNibble(prefix *net.IPNet, size int) nibble {
	var n []byte

	if size < 48 {
		n = []byte(prefix.IP)[4:8]
	} else {
		n = []byte(prefix.IP)[6:8]
	}
	return nibble(n)
}
