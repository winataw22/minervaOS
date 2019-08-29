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

func requestAllocation(farm string, store *allocationStore) (*net.IPNet, *net.IPNet, error) {
	store.Lock()
	defer store.Unlock()
	farmAlloc, ok := store.Allocations[farm]
	if !ok {
		return nil, nil, fmt.Errorf("farm %s does not have a prefix registered", farm)
	}

	newAlloc, err := allocate(farmAlloc)
	if err != nil {
		return nil, nil, err
	}

	return newAlloc, farmAlloc.Allocation, nil
}

func getNetworkZero(farm string, store *allocationStore) (*net.IPNet, int, error) {
	store.Lock()
	defer store.Unlock()
	farmAlloc, ok := store.Allocations[farm]
	if !ok {
		return nil, 0, fmt.Errorf("farm %s does not have a prefix registered", farm)
	}

	ipv6net, err := netaddr.ParseIPv6Net(farmAlloc.Allocation.String())
	if err != nil {
		return nil, 0, err
	}
	subnet := ipv6net.NthSubnet(64, 0)
	allocSize, _ := farmAlloc.Allocation.Mask.Size()
	return convert(subnet), allocSize, nil
}

func allocate(allocation *allocation) (*net.IPNet, error) {
	ipv6net, err := netaddr.ParseIPv6Net(allocation.Allocation.String())
	if err != nil {
		return nil, err
	}

	subnetCount := ipv6net.SubnetCount(64)
	if uint64(len(allocation.SubNetUsed)) >= subnetCount {
		return nil, fmt.Errorf("all subnets already allocated")
	}

	// random from 000f to subnetCount
	// we never hand out the network 0 to f cause we keep it for
	// adminstrative purposes (routing segment, mgmt, tunnel sources... )
	nth := rand.Int63n(int64(subnetCount)-16) + 16
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
