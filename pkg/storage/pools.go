package storage

import (
	"slices"

	"github.com/threefoldtech/zos/pkg/kernel"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

// utils for pool ordering and presence

// Policy needed
type Policy func(s *Module) []filesystem.Pool

func poolCmp(a, b filesystem.Pool) int {
	_, errA := a.Mounted()
	_, errB := b.Mounted()

	// if the 2 pools has the same mount state, then both can
	// be 0
	if errA != nil && errB != nil || errA == nil && errB == nil {
		return 0
	} else if errA == nil {
		// mounted pool comes first
		return -1
	} else {
		return 1
	}
}

func PolicySSDOnly(s *Module) []filesystem.Pool {
	slices.SortFunc(s.ssds, poolCmp)
	return s.ssds
}

func PolicyHDDOnly(s *Module) []filesystem.Pool {
	slices.SortFunc(s.hdds, poolCmp)
	return s.hdds
}

func PolicySSDFirst(s *Module) []filesystem.Pool {
	pools := PolicySSDOnly(s)

	// if missing ssd is supported, this policy
	// will also use the hdd pools for provisioning
	// and cache
	if kernel.GetParams().Exists(kernel.MissingSSD) {
		pools = append(pools, PolicyHDDOnly(s)...)
	}

	return pools
}

// get available pools in defined presence
func (s *Module) pools(policy Policy) []filesystem.Pool {
	return policy(s)
}
