package capacity

import (
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/capacity/dmi"
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
	storage modules.StorageModule
}

// NewResourceOracle creates a new ResourceOracle
func NewResourceOracle(s modules.StorageModule) *ResourceOracle {
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
