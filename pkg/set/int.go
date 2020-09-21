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

// UIntSet is a set containing uint
type UIntSet struct {
	sync.RWMutex
	m map[uint]struct{}
}

// NewInt creates a new set for int
func NewInt() *UIntSet {
	return &UIntSet{
		m: make(map[uint]struct{}),
	}
}

// Add tries to add port to the set. If port is already
// present errPortConflict is return otherwise nil is returned
func (p *UIntSet) Add(i uint) error {
	p.Lock()
	defer p.Unlock()

	if _, exist := p.m[i]; exist {
		return ErrConflict{Port: i}
	}
	p.m[i] = struct{}{}
	return nil
}

// Remove removes a port from the set
// removes never fails cause if the port is not in the set
// remove is a nop-op
func (p *UIntSet) Remove(i uint) {
	p.Lock()
	defer p.Unlock()

	delete(p.m, i)
}

// List returns a list of uint present in the set
func (p *UIntSet) List() ([]uint, error) {
	p.RLock()
	defer p.RUnlock()

	l := make([]uint, 0, len(p.m))
	for i := range p.m {
		l = append(l, i)
	}
	return l, nil
}
