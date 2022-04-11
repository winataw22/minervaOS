package pkg

//go:generate mkdir -p stubs

//go:generate zbusc -module registrar -version 0.0.1 -name registrar -package stubs github.com/threefoldtech/zos/pkg+Registrar stubs/registrar_stub.go

type Registrar interface {
	NodeID() (uint32, error)
	TwinID() (uint32, error)
}
