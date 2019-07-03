package main

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
		_, allocation, err := net.ParseCIDR(tc.Alloc)
		require.NoError(t, err)
		alloc := &Allocation{Allocation: allocation}
		subnet, err := allocate(alloc)
		require.NoError(t, err)
		fmt.Println(subnet.String())
	}
}

func TestNetIP(t *testing.T) {
	for _, tc := range []struct {
		netZero   string
		alloc     string
		allocSize int
		result    string
	}{
		{
			netZero:   "2a02:2788:cc02::",
			alloc:     "2a02:2788:cc02:abcd::",
			allocSize: 48,
			result:    "2a02:2788:cc02::abcd",
		},
		{
			allocSize: 32,
			netZero:   "2a02:2788::",
			alloc:     "2a02:2788:cc02:abcd::",
			result:    "2a02:2788::cc02:abcd",
		},
	} {

		t.Run(fmt.Sprintf("%d", tc.allocSize), func(t *testing.T) {
			netZero := &net.IPNet{
				IP:   net.ParseIP(tc.netZero),
				Mask: net.CIDRMask(64, 128),
			}
			alloc := &net.IPNet{
				IP:   net.ParseIP(tc.alloc),
				Mask: net.CIDRMask(64, 128),
			}

			ip := netZeroIP(netZero, alloc, tc.allocSize)
			assert.Equal(t, net.ParseIP(tc.result), ip)
		})
	}
}
