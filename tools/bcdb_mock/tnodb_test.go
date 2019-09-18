package main

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixAllocation(t *testing.T) {
	for _, tc := range []struct {
		Alloc string
	}{
		{"2a02:2788:0000::/48"},
		{"2a02:2788:1000::/48"},
		{"2a02:2788:0100::/48"},
	} {
		_, a, err := net.ParseCIDR(tc.Alloc)
		require.NoError(t, err)
		alloc := &allocation{Allocation: a}
		subnet, err := allocate(alloc)
		require.NoError(t, err)
		fmt.Println(subnet.String())
	}
}
