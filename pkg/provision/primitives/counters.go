package primitives

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Counter interface
type Counter interface {
	// Increment counter atomically by v
	Increment(v uint64) uint64
	// Decrement counter atomically by v
	Decrement(v uint64) uint64
	// Current returns the current value
	Current() uint64
}

// CounterUint64 value for safe increment/decrement
type CounterUint64 uint64

// Increment counter atomically by one
func (c *CounterUint64) Increment(v uint64) uint64 {
	return atomic.AddUint64((*uint64)(c), v)
}

// Decrement counter atomically by one
func (c *CounterUint64) Decrement(v uint64) uint64 {
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
func (c *CounterUint64) Current() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}

// Counters tracks the amount of primitives workload deployed and
// the amount of resource unit used
type Counters struct {
	containers CounterUint64
	volumes    CounterUint64
	networks   CounterUint64
	zdbs       CounterUint64
	vms        CounterUint64

	SRU CounterUint64 // SSD storage in bytes
	HRU CounterUint64 // HDD storage in bytes
	MRU CounterUint64 // Memory storage in bytes
	CRU CounterUint64 // CPU count absolute
}

// CurrentWorkloads return the number of each workloads provisioned on the system
func (c *Counters) CurrentWorkloads() directory.WorkloadAmount {
	return directory.WorkloadAmount{
		Network:      uint16(c.networks.Current()),
		Volume:       uint16(c.volumes.Current()),
		ZDBNamespace: uint16(c.zdbs.Current()),
		Container:    uint16(c.containers.Current()),
		K8sVM:        uint16(c.vms.Current()),
	}
}

// CurrentUnits return the number of each resource units reserved on the system
func (c *Counters) CurrentUnits() directory.ResourceAmount {
	gib := float64(gib)
	return directory.ResourceAmount{
		Cru: c.CRU.Current(),
		Mru: float64(c.MRU.Current()) / gib,
		Hru: float64(c.HRU.Current()) / gib,
		Sru: float64(c.SRU.Current()) / gib,
	}
}

const (
	mib = uint64(1024 * 1024)
	gib = uint64(mib * 1024)
)

// Increment is called by the provision.Engine when a reservation has been provisionned
func (c *Counters) Increment(r *provision.Reservation) error {

	var (
		u   resourceUnits
		err error
	)

	switch r.Type {
	case VolumeReservation:
		c.volumes.Increment(1)
		u, err = processVolume(r)
	case ContainerReservation:
		c.containers.Increment(1)
		u, err = processContainer(r)
	case ZDBReservation:
		c.zdbs.Increment(1)
		u, err = processZdb(r)
	case KubernetesReservation:
		c.vms.Increment(1)
		u, err = processKubernetes(r)
	case NetworkReservation, NetworkResourceReservation:
		c.networks.Increment(1)
		u = resourceUnits{}
		err = nil
	default:
		u = resourceUnits{}
		err = nil
	}
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
func (c *Counters) Decrement(r *provision.Reservation) error {

	var (
		u   resourceUnits
		err error
	)

	switch r.Type {
	case VolumeReservation:
		c.volumes.Decrement(1)
		u, err = processVolume(r)
	case ContainerReservation:
		c.containers.Decrement(1)
		u, err = processContainer(r)
	case ZDBReservation:
		c.zdbs.Decrement(1)
		u, err = processZdb(r)
	case KubernetesReservation:
		c.vms.Decrement(1)
		u, err = processKubernetes(r)
	case NetworkReservation, NetworkResourceReservation:
		c.networks.Decrement(1)
		u = resourceUnits{}
		err = nil
	default:
		u = resourceUnits{}
		err = nil
	}
	if err != nil {
		return err
	}

	c.CRU.Decrement(u.CRU)
	c.MRU.Decrement(u.MRU)
	c.SRU.Decrement(u.SRU)
	c.HRU.Decrement(u.HRU)

	return nil
}

type resourceUnits struct {
	SRU uint64 `json:"sru,omitempty"`
	HRU uint64 `json:"hru,omitempty"`
	MRU uint64 `json:"mru,omitempty"`
	CRU uint64 `json:"cru,omitempty"`
}

func processVolume(r *provision.Reservation) (u resourceUnits, err error) {
	var volume Volume
	if err = json.Unmarshal(r.Data, &volume); err != nil {
		return u, err
	}

	// volume.size and SRU is in GiB
	switch volume.Type {
	case pkg.SSDDevice:
		u.SRU = volume.Size * gib
	case pkg.HDDDevice:
		u.HRU = volume.Size * gib
	}

	return u, nil
}

func processContainer(r *provision.Reservation) (u resourceUnits, err error) {
	var cont Container
	if err = json.Unmarshal(r.Data, &cont); err != nil {
		return u, err
	}
	u.CRU = uint64(cont.Capacity.CPU)
	// memory is in MiB
	u.MRU = cont.Capacity.Memory * mib
	if cont.Capacity.DiskType == pkg.SSDDevice {
		u.SRU = cont.Capacity.DiskSize * mib
	} else if cont.Capacity.DiskType == pkg.HDDDevice {
		u.HRU = cont.Capacity.DiskSize * mib
	}

	return u, nil
}

func processZdb(r *provision.Reservation) (u resourceUnits, err error) {
	if r.Type != ZDBReservation {
		return u, fmt.Errorf("wrong type or reservation %s, excepted %s", r.Type, ZDBReservation)
	}
	var zdbVolume ZDB
	if err := json.Unmarshal(r.Data, &zdbVolume); err != nil {
		return u, err
	}

	switch zdbVolume.DiskType {
	case pkg.SSDDevice:
		u.SRU = zdbVolume.Size * gib
	case pkg.HDDDevice:
		u.HRU = zdbVolume.Size * gib
	}

	return u, nil
}

func processKubernetes(r *provision.Reservation) (u resourceUnits, err error) {
	var k8s Kubernetes
	if err = json.Unmarshal(r.Data, &k8s); err != nil {
		return u, err
	}

	// size are defined at https://github.com/threefoldtech/zos/blob/master/pkg/provision/kubernetes.go#L311
	switch k8s.Size {
	case 1:
		u.CRU = 1
		u.MRU = 2 * gib
		u.SRU = 50 * gib
	case 2:
		u.CRU = 2
		u.MRU = 4 * gib
		u.SRU = 100 * gib
	case 3:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 25 * gib
	case 4:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 50 * gib
	case 5:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 200 * gib
	case 6:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 50 * gib
	case 7:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 100 * gib
	case 8:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 400 * gib
	case 9:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 100 * gib
	case 10:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 200 * gib
	case 11:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 800 * gib
	case 12:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 200 * gib
	case 13:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 400 * gib
	case 14:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 800 * gib
	case 15:
		u.CRU = 1
		u.MRU = 2 * gib
		u.SRU = 25 * gib
	case 16:
		u.CRU = 2
		u.MRU = 4 * gib
		u.SRU = 50 * gib
	case 17:
		u.CRU = 4
		u.MRU = 8 * gib
		u.SRU = 50 * gib
	}

	return u, nil
}
