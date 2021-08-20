package ifaceutil

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"
)

func TestHardwareAddrFromInputBytes_Simple_Nil(t *testing.T) {
	addr := HardwareAddrFromInputBytes(nil)
	expectedAddr := net.HardwareAddr{0xd6, 0x1d, 0x8c, 0xd9, 0x8f, 0x00}
	if !bytes.Equal(addr[:], expectedAddr[:]) {
		t.Fatalf("%v != %v", addr, expectedAddr)
	}
}

func TestHardwareAddrFromInputBytes_Simple_42(t *testing.T) {
	addr := HardwareAddrFromInputBytes([]byte{0x42})
	expectedAddr := net.HardwareAddr{0x9e, 0x5e, 0xd6, 0x78, 0xfe, 0x57}
	if !bytes.Equal(addr[:], expectedAddr[:]) {
		t.Fatalf("%v != %v", addr, expectedAddr)
	}
}

func TestHardwareAddrFromInputBytes_Simple_FromNodeID(t *testing.T) {
	const nodeID = "A34YUGenHKyhjDMAUKZe4cVDtJM2wQ4n4XRkfGUUEYdy"
	addr := HardwareAddrFromInputBytes([]byte(nodeID[:]))
	expectedAddr := net.HardwareAddr{0x8a, 0xd3, 0x36, 0x10, 0x7e, 0xe9}
	if !bytes.Equal(addr[:], expectedAddr[:]) {
		t.Fatalf("%v != %v", addr, expectedAddr)
	}
}

func TestHardwareAddrFromInputBytes_Range(t *testing.T) {
	for n := 0; n < 1024; n++ {
		for s := 0; s < 32; s++ {
			b := make([]byte, s)
			_, _ = rand.Read(b[:])
			HardwareAddrFromInputBytes(b)
		}
	}
}

func BenchmarkHardwareAddrFromInputBytes_Range(b *testing.B) {
	var bs [33]byte
	_, _ = rand.Read(bs[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HardwareAddrFromInputBytes(bs[:])
	}
}
