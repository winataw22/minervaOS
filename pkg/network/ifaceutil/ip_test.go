package ifaceutil

import (
	"crypto/rand"
	"testing"
)

func TestIPv6SuffixFromInputBytesAsHex_7_Simple_Nil(t *testing.T) {
	suffix := IPv6SuffixFromInputBytesAsHex(nil, 7)
	const expectedSuffix = "d4:1d8c:d98f:00b2"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_8_Simple_Nil(t *testing.T) {
	suffix := IPv6SuffixFromInputBytesAsHex(nil, 8)
	const expectedSuffix = "d41d:8cd9:8f00:b204"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_7_Simple_42(t *testing.T) {
	suffix := IPv6SuffixFromInputBytesAsHex([]byte{0x42}, 7)
	const expectedSuffix = "9d:5ed6:78fe:57bc"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_8_Simple_42(t *testing.T) {
	suffix := IPv6SuffixFromInputBytesAsHex([]byte{0x42}, 8)
	const expectedSuffix = "9d5e:d678:fe57:bcca"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_7_Simple_NodeID(t *testing.T) {
	const nodeID = "A34YUGenHKyhjDMAUKZe4cVDtJM2wQ4n4XRkfGUUEYdy"
	suffix := IPv6SuffixFromInputBytesAsHex([]byte(nodeID[:]), 7)
	const expectedSuffix = "8b:d336:107e:e915"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_8_Simple_NodeID(t *testing.T) {
	const nodeID = "A34YUGenHKyhjDMAUKZe4cVDtJM2wQ4n4XRkfGUUEYdy"
	suffix := IPv6SuffixFromInputBytesAsHex([]byte(nodeID[:]), 8)
	const expectedSuffix = "8bd3:3610:7ee9:1507"
	if suffix != expectedSuffix {
		t.Fatalf("%s != %s", suffix, expectedSuffix)
	}
}

func TestIPv6SuffixFromInputBytesAsHex_1_Range(t *testing.T) {
	for n := 0; n < 1024; n++ {
		for s := 0; s < 32; s++ {
			b := make([]byte, s)
			rand.Read(b[:])
			IPv6SuffixFromInputBytesAsHex(b, 1)
		}
	}
}

func TestIPv6SuffixFromInputBytesAsHex_7_Range(t *testing.T) {
	for n := 0; n < 1024; n++ {
		for s := 0; s < 32; s++ {
			b := make([]byte, s)
			rand.Read(b[:])
			IPv6SuffixFromInputBytesAsHex(b, 7)
		}
	}
}

func TestIPv6SuffixFromInputBytesAsHex_8_Range(t *testing.T) {
	for n := 0; n < 1024; n++ {
		for s := 0; s < 32; s++ {
			b := make([]byte, s)
			rand.Read(b[:])
			IPv6SuffixFromInputBytesAsHex(b, 8)
		}
	}
}

func BenchmarkIPv6SuffixFromInputBytesAsHex_7_Range(b *testing.B) {
	var bs [33]byte
	rand.Read(bs[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IPv6SuffixFromInputBytesAsHex(bs[:], 7)
	}
}

func BenchmarkIPv6SuffixFromInputBytesAsHex_8_Range(b *testing.B) {
	var bs [33]byte
	rand.Read(bs[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IPv6SuffixFromInputBytesAsHex(bs[:], 8)
	}
}

func BenchmarkIPv6SuffixFromInputBytesAsHex_12_Range(b *testing.B) {
	var bs [33]byte
	rand.Read(bs[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IPv6SuffixFromInputBytesAsHex(bs[:], 12)
	}
}
