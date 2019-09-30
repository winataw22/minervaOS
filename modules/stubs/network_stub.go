package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	modules "github.com/threefoldtech/zosv2/modules"
)

type NetworkerStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewNetworkerStub(client zbus.Client) *NetworkerStub {
	return &NetworkerStub{
		client: client,
		module: "network",
		object: zbus.ObjectID{
			Name:    "network",
			Version: "0.0.1",
		},
	}
}

func (s *NetworkerStub) Addrs(arg0 string, arg1 string) (ret0 [][]uint8, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Addrs", args...)
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

func (s *NetworkerStub) CreateNR(arg0 modules.Network) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "CreateNR", args...)
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

func (s *NetworkerStub) DeleteNR(arg0 modules.Network) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "DeleteNR", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) Join(arg0 modules.NetID, arg1 string, arg2 []string) (ret0 modules.Member, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.Request(s.module, s.object, "Join", args...)
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

func (s *NetworkerStub) ZDBPrepare() (ret0 string, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "ZDBPrepare", args...)
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
