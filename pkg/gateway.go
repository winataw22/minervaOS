package pkg

//go:generate mkdir -p stubs

//go:generate zbusc -module gateway -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+Gateway stubs/gateway_stub.go

type Gateway interface {
	SetNamedProxy(wlID string, prefix string, backends []string) (string, error)
	DeleteNamedProxy(wlID string) error
}
