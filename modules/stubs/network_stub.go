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

func (s *NetworkerStub) ApplyNetResource(arg0 modules.Network) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "ApplyNetResource", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) DeleteNetResource(arg0 modules.Network) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "DeleteNetResource", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GenerateWireguarKeyPair(arg0 modules.NetID) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "GenerateWireguarKeyPair", args...)
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

func (s *NetworkerStub) GetNetwork(arg0 modules.NetID) (ret0 modules.Network, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "GetNetwork", args...)
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

func (s *NetworkerStub) JoinNetwork(arg0 modules.NetID, arg1 uint16, arg2 string) (ret0 modules.Network, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.Request(s.module, s.object, "JoinNetwork", args...)
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

func (s *NetworkerStub) PublishWGPubKey(arg0 string, arg1 modules.NetID) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "PublishWGPubKey", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
