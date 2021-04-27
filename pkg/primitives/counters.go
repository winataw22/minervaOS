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

	SRU  AtomicValue // SSD storage in bytes
	HRU  AtomicValue // HDD storage in bytes
	MRU  AtomicValue // Memory storage in bytes
	CRU  AtomicValue // CPU count absolute
	IPv4 AtomicValue // IPv4 count absolute
}

const (
	mib = uint64(1024 * 1024)
	gib = uint64(mib * 1024)
)

// Increment is called by the provision.Engine when a reservation has been provisionned
func (c *Counters) Increment(cap gridtypes.Capacity) {
	c.CRU.Increment(cap.CRU)
	c.MRU.Increment(cap.MRU)
	c.SRU.Increment(cap.SRU)
	c.HRU.Increment(cap.HRU)
	c.IPv4.Increment(cap.IPV4U)
}

// Decrement is called by the provision.Engine when a reservation has been decommissioned
func (c *Counters) Decrement(cap gridtypes.Capacity) {
	c.CRU.Decrement(cap.CRU)
	c.MRU.Decrement(cap.MRU)
	c.SRU.Decrement(cap.SRU)
	c.HRU.Decrement(cap.HRU)
	c.IPv4.Decrement(cap.IPV4U)
}
