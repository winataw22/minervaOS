package pkg

import (
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs

//go:generate zbusc -module qsfsd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+QSFSD stubs/qsfsd_stub.go

type QSFSMetrics struct {
	Consumption map[string]NetMetric
}

type QSFSInfo struct {
	Path            string
	MetricsEndpoint string
}

func (q *QSFSMetrics) Nu(wlID string) (result uint64) {
	if v, ok := q.Consumption[wlID]; ok {
		result += v.NetRxBytes
		result += v.NetTxBytes
	}
	return
}

type QSFSD interface {
	Mount(wlID string, cfg zos.QuantumSafeFS) (QSFSInfo, error)
	UpdateMount(wlID string, cfg zos.QuantumSafeFS) (QSFSInfo, error)
	Unmount(wlID string) error
	Metrics() (QSFSMetrics, error)
}
