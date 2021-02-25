package primitives

import (
	"sync/atomic"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// AtomicValue value for safe increment/decrement
type AtomicValue uint64

// Increment counter atomically by one
func (c *AtomicValue) Increment(v uint64) uint64 {
	return atomic.AddUint64((*uint64)(c), v)
}

// Decrement counter atomically by one
func (c *AtomicValue) Decrement(v uint64) uint64 {
	// spinlock until the decrement succeeds
	for {
		current := c.Current()
		// make sure we don't decrement below 0
		dec := v
		if dec > current {
			dec = current
		}
		// compute new value
		n := current - dec
		// only swap if `current`, and therefore the above calculations,
		// are still valid
		if atomic.CompareAndSwapUint64((*uint64)(c), current, n) {
			return n
		}
	}
}

// Current returns the current value
func (c *AtomicValue) Current() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}

// Counters tracks the amount of primitives workload deployed and
// the amount of resource unit used
type Counters struct {
	//types map[gridtypes.WorkloadType]AtomicValue

	SRU AtomicValue // SSD storage in bytes
	HRU AtomicValue // HDD storage in bytes
	MRU AtomicValue // Memory storage in bytes
	CRU AtomicValue // CPU count absolute
}

const (
	mib = uint64(1024 * 1024)
	gib = uint64(mib * 1024)
)

// Increment is called by the provision.Engine when a reservation has been provisionned
func (c *Counters) Increment(r *gridtypes.Workload) error {
	u, err := r.Capacity()
	if err != nil {
		return err
	}

	c.CRU.Increment(u.CRU)
	c.MRU.Increment(u.MRU)
	c.SRU.Increment(u.SRU)
	c.HRU.Increment(u.HRU)

	return nil
}

// Decrement is called by the provision.Engine when a reservation has been decommissioned
func (c *Counters) Decrement(r *gridtypes.Workload) error {
	u, err := r.Capacity()
	if err != nil {
		return err
	}

	c.CRU.Decrement(u.CRU)
	c.MRU.Decrement(u.MRU)
	c.SRU.Decrement(u.SRU)
	c.HRU.Decrement(u.HRU)

	return nil
}
