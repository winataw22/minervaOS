package pkg

//go:generate zbusc -module provision -version 0.0.1 -name provision -package stubs github.com/threefoldtech/zos/pkg+Provision stubs/provision_stub.go
//go:generate zbusc -module provision -version 0.0.1 -name statistics -package stubs github.com/threefoldtech/zos/pkg+Statistics stubs/statistics_stub.go

import (
	"context"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Provision interface
type Provision interface {
	DecommissionCached(id string, reason string) error
	// GetWorkloadStatus: returns status, bool(true if workload exits otherwise it is false), error
	GetWorkloadStatus(id string) (gridtypes.ResultState, bool, error)
}

type Statistics interface {
	ReservedStream(ctx context.Context) <-chan gridtypes.Capacity
	Current() (gridtypes.Capacity, error)
	Total() gridtypes.Capacity
	Workloads() (int, error)
}
