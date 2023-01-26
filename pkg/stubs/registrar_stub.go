// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
)

type RegistrarStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewRegistrarStub(client zbus.Client) *RegistrarStub {
	return &RegistrarStub{
		client: client,
		module: "registrar",
		object: zbus.ObjectID{
			Name:    "registrar",
			Version: "0.0.1",
		},
	}
}

func (s *RegistrarStub) NodeID(ctx context.Context) (ret0 uint32, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "NodeID", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarStub) TwinID(ctx context.Context) (ret0 uint32, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "TwinID", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}
