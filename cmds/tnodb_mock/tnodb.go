package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/dspinhirne/netaddr-go"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func requestAllocation(farm string, store *allocationStore) (*net.IPNet, error) {
	store.Lock()
	defer store.Unlock()
	farmAlloc, ok := store.Allocations[farm]
	if !ok {
		return nil, fmt.Errorf("farm %s does not have a prefix registered", farm)
	}

	return allocate(farmAlloc)
}

func allocate(allocation *Allocation) (*net.IPNet, error) {
	ipv6net, err := netaddr.ParseIPv6Net(allocation.Allocation.String())
	if err != nil {
		panic(err)
	}

	subnetCount := ipv6net.SubnetCount(64)
	if uint64(len(allocation.SubNetUsed)) >= subnetCount {
		return nil, fmt.Errorf("all subnets already allocated")
	}
	nth := rand.Int63n(int64(subnetCount))
	for {
		if !isIn(nth, allocation.SubNetUsed) {
			allocation.SubNetUsed = append(allocation.SubNetUsed, uint64(nth))
			break
		}
		nth = rand.Int63n(int64(subnetCount))
	}

	subnet := ipv6net.NthSubnet(64, uint64(nth))
	return convert(subnet), nil

}

// FIXME: use someting better then O(n)
func isIn(target int64, list []uint64) bool {
	for _, x := range list {
		if uint64(target) == x {
			return true
		}
	}
	return false
}

// FIXME: avoid passing by string representation to convert
func convert(subnet *netaddr.IPv6Net) *net.IPNet {
	_, net, err := net.ParseCIDR(subnet.String())
	if err != nil {
		panic(err)
	}
	return net
}
