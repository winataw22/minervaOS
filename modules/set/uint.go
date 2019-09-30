package set

import (
	"fmt"
	"sync"
)

// ErrConflict is return when trying to add a port
// in the set that is already present
type ErrConflict struct {
	Port uint
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("port %d is already in the set", e.Port)
}

// UintSet is a set containing uint
type UintSet struct {
	sync.RWMutex
	m map[uint]struct{}
}

// NewUint creates a new set for uint
func NewUint() *UintSet {
	return &UintSet{
		m: make(map[uint]struct{}),
	}
}

// Add tries to add port to the set. If port is already
// present errPortConflict is return otherwise nil is returned
func (p *UintSet) Add(i uint) error {
	p.Lock()
	defer p.Unlock()

	_, exists := p.m[i]
	if exists {
		return ErrConflict{i}
	}
	p.m[i] = struct{}{}
	return nil
}

// Remove removes a port from the set
// removes never fails cause if the port is not in the set
// remove is a nop-op
func (p *UintSet) Remove(i uint) {
	p.Lock()
	defer p.Unlock()

	_, exists := p.m[i]
	if exists {
		delete(p.m, i)
	}
}

// List returns a list of uint present in the set
func (p *UintSet) List() []uint {
	p.RLock()
	defer p.RUnlock()

	l := make([]uint, 0, len(p.m))
	for i := range p.m {
		l = append(l, i)
	}
	return l
}
