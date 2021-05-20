package primitives

import (
	"sync/atomic"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// AtomicUnit value for safe increment/decrement
type AtomicUnit gridtypes.Unit

// Increment counter atomically by one
func (c *AtomicUnit) Increment(v gridtypes.Unit) gridtypes.Unit {
	return gridtypes.Unit(atomic.AddUint64((*uint64)(c), uint64(v)))
}

// Decrement counter atomically by one
func (c *AtomicUnit) Decrement(v gridtypes.Unit) gridtypes.Unit {
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
		if atomic.CompareAndSwapUint64((*uint64)(c), uint64(current), uint64(n)) {
			return gridtypes.Unit(n)
		}
	}
}

// Current returns the current value
func (c *AtomicUnit) Current() gridtypes.Unit {
	return gridtypes.Unit(atomic.LoadUint64((*uint64)(c)))
}

// Counters tracks the amount of primitives workload deployed and
// the amount of resource unit used
type Counters struct {
	//types map[gridtypes.WorkloadType]AtomicValue

	SRU  AtomicUnit // SSD storage in bytes
	HRU  AtomicUnit // HDD storage in bytes
	MRU  AtomicUnit // Memory storage in bytes
	CRU  AtomicUnit // CPU count absolute
	IPv4 AtomicUnit // IPv4 count absolute
}

// Increment is called by the provision.Engine when a reservation has been provisionned
func (c *Counters) Increment(cap gridtypes.Capacity) {
	c.CRU.Increment(gridtypes.Unit(cap.CRU))
	c.MRU.Increment(cap.MRU)
	c.SRU.Increment(cap.SRU)
	c.HRU.Increment(cap.HRU)
	c.IPv4.Increment(gridtypes.Unit(cap.IPV4U))
}

// Decrement is called by the provision.Engine when a reservation has been decommissioned
func (c *Counters) Decrement(cap gridtypes.Capacity) {
	c.CRU.Decrement(gridtypes.Unit(cap.CRU))
	c.MRU.Decrement(cap.MRU)
	c.SRU.Decrement(cap.SRU)
	c.HRU.Decrement(cap.HRU)
	c.IPv4.Decrement(gridtypes.Unit(cap.IPV4U))
}
