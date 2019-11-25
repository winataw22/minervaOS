package ifaceutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
)

// IPv6SuffixFromInputBytes returns a deterministic IPv6 suffix
// for a given byte slice, with n equals the amount of bytes the suffix has.
// n has to be within the range of [0, 16].
func IPv6SuffixFromInputBytes(b []byte, n int) []byte {
	if n <= 0 {
		return nil
	}
	if n > 16 {
		panic(fmt.Sprintf("up to 16 bytes are allowed, %d is an invalid amount of bytes", n))
	}
	hs := md5.Sum(b[:])
	return hs[:n]
}

// IPv6SuffixFromInputBytesAsHex returns a deterministic IPv6 suffix hex-encoded
// for a given byte slice, with n equals the amount of bytes the suffix has.
// n has to be within the range of [0, 16].
func IPv6SuffixFromInputBytesAsHex(b []byte, n int) string {
	ob := IPv6SuffixFromInputBytes(b, n)
	const (
		maxLen   = len("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
		hexDigit = "0123456789abcdef"
	)
	out := make([]byte, 0, maxLen)
	gs := 0
	if len(ob)%2 == 1 {
		gs = 1
	}
	for _, o := range ob {
		out = append(out, hexDigit[o>>4], hexDigit[o&0xF])
		gs++
		if gs == 2 {
			out = append(out, ':')
			gs = 0
		}
	}
	return string(bytes.TrimSuffix(out, []byte{':'}))
}
