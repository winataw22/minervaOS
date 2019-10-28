package capacity

import (
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

// Capacity hold the amount of resource unit of a node
type Capacity struct {
	CRU uint64 `json:"cru"`
	MRU uint64 `json:"mru"`
	SRU uint64 `json:"sru"`
	HRU uint64 `json:"hru"`
}

// ResourceOracle is the structure responsible for capacity tracking
type ResourceOracle struct {
	storage pkg.StorageModule
}

// NewResourceOracle creates a new ResourceOracle
func NewResourceOracle(s pkg.StorageModule) *ResourceOracle {
	return &ResourceOracle{storage: s}
}

// Total returns the total amount of resource units of the node
func (r *ResourceOracle) Total() (c *Capacity, err error) {
	c = &Capacity{}

	c.CRU, err = r.cru()
	if err != nil {
		return nil, err
	}
	c.MRU, err = r.mru()
	if err != nil {
		return nil, err
	}
	c.SRU, err = r.sru()
	if err != nil {
		return nil, err
	}
	c.HRU, err = r.hru()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// DMI run and parse dmidecode commands
func (r *ResourceOracle) DMI() (*dmi.DMI, error) {
	return dmi.Decode()
}

// Uptime returns the uptime of the node
func (r *ResourceOracle) Uptime() (uint64, error) {
	info, err := host.Info()
	if err != nil {
		return 0, err
	}
	return info.Uptime, nil
}
