package portm

import (
	"errors"

	"github.com/threefoldtech/zos/pkg/network/portm/backend"
)

// ErrNoFreePort is returned when trying to reserve a port but all the
// the port of the range have been already reserved
var ErrNoFreePort = errors.New("no free port find")

// PortRange hold the beginging and end of a range of port
// a PortAllocator can reserve
type PortRange struct {
	Start int
	End   int
}

// Allocator implements the PortAllocator interface
type Allocator struct {
	pRange PortRange
	store  backend.Store
}

var _ PortAllocator = (*Allocator)(nil)

// NewAllocator return a PortAllocator
func NewAllocator(pRange PortRange, store backend.Store) *Allocator {
	return &Allocator{
		pRange: pRange,
		store:  store,
	}
}

// Reserve implements PortAllocator interface
func (a *Allocator) Reserve(ns string) (int, error) {
	if err := a.store.Lock(); err != nil {
		return 0, err
	}
	defer func() {
		_ = a.store.Unlock()
	}()

	allocatedPorts, err := a.store.GetByNS(ns)
	if err != nil {
		return 0, err
	}

	start, err := a.store.LastReserved(ns)
	if err != nil {
		return 0, err
	}

	if start == -1 || start >= a.pRange.End {
		// nothing reserve yet
		// or we reach the end of the range
		// then start from the start to re-use released port
		start = a.pRange.Start
	}

	for port := start; port <= a.pRange.End; port++ {
		if contains(allocatedPorts, port) {
			continue
		}

		reserved, err := a.store.Reserve(ns, port)
		if err != nil {
			return 0, err
		}
		if reserved {
			return port, nil
		}
	}

	return 0, ErrNoFreePort
}

// Release implements PortAllocator interface
func (a *Allocator) Release(ns string, port int) error {
	if err := a.store.Lock(); err != nil {
		return err
	}
	defer func() {
		_ = a.store.Unlock()
	}()

	return a.store.Release(ns, port)
}

func contains(s []int, port int) bool {
	for i := range s {
		if s[i] == port {
			return true
		}
	}
	return false
}
