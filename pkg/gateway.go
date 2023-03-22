package pkg

import "github.com/threefoldtech/zos/pkg/gridtypes/zos"

//go:generate mkdir -p stubs

//go:generate zbusc -module gateway -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+Gateway stubs/gateway_stub.go

type GatewayMetrics struct {
	Request  map[string]float64
	Response map[string]float64
}

func (m *GatewayMetrics) Nu(service string) (result uint64) {
	if v, ok := m.Request[service]; ok {
		result += uint64(v)
	}

	if v, ok := m.Response[service]; ok {
		result += uint64(v)
	}

	return
}

type Gateway interface {
	SetNamedProxy(wlID string, config zos.GatewayNameProxy) (string, error)
	SetFQDNProxy(wlID string, config zos.GatewayFQDNProxy) error
	DeleteNamedProxy(wlID string) error
	Metrics() (GatewayMetrics, error)
}
