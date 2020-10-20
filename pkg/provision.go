package pkg

//go:generate zbusc -module provision -version 0.0.1 -name provision -package stubs github.com/threefoldtech/zos/pkg+Provision stubs/provision_stub.go

import "context"

// ProvisionCounters struct
type ProvisionCounters struct {
	Container int64 `json:"container"`
	Volume    int64 `jons:"volume"`
	Network   int64 `json:"network"`
	ZDB       int64 `json:"zdb"`
	VM        int64 `json:"vm"`
	Debug     int64 `json:"debug"`
}

// Provision interface
type Provision interface {
	Counters(ctx context.Context) <-chan ProvisionCounters
	DecommissionCached(id string, reason string) error
}
