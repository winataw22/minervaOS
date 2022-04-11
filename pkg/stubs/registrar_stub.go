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
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	ret1 = new(zbus.RemoteError)
	if err := result.Unmarshal(1, &ret1); err != nil {
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
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	ret1 = new(zbus.RemoteError)
	if err := result.Unmarshal(1, &ret1); err != nil {
		panic(err)
	}
	return
}
