package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	modules "github.com/threefoldtech/zosv2/modules"
)

type ZDBAllocaterStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewZDBAllocaterStub(client zbus.Client) *ZDBAllocaterStub {
	return &ZDBAllocaterStub{
		client: client,
		module: "storage",
		object: zbus.ObjectID{
			Name:    "storage",
			Version: "0.0.1",
		},
	}
}

func (s *ZDBAllocaterStub) Allocate(arg0 modules.DeviceType, arg1 uint64, arg2 modules.ZDBMode) (ret0 string, ret1 string, ret2 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.Request(s.module, s.object, "Allocate", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	if err := result.Unmarshal(1, &ret1); err != nil {
		panic(err)
	}
	ret2 = new(zbus.RemoteError)
	if err := result.Unmarshal(2, &ret2); err != nil {
		panic(err)
	}
	return
}

func (s *ZDBAllocaterStub) Claim(arg0 string, arg1 uint64) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Claim", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
