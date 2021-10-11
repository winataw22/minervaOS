package pkg

//go:generate mkdir -p stubs

//go:generate zbusc -module gateway -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+Gateway stubs/gateway_stub.go

type GatewayMetrics struct {
	Sent     map[string]float64
	Received map[string]float64
}

func (m *GatewayMetrics) Nu(service string) (result uint64) {
	if v, ok := m.Sent[service]; ok {
		result += uint64(v)
	}

	if v, ok := m.Received[service]; ok {
		result += uint64(v)
	}

	return
}

type Gateway interface {
	SetNamedProxy(wlID string, prefix string, backends []string, TLSPassthrough bool, twinID uint32) (string, error)
	SetFQDNProxy(wlID string, fqdn string, backends []string, TLSPassthrough bool) error
	DeleteNamedProxy(wlID string) error
	Metrics() (GatewayMetrics, error)
}
