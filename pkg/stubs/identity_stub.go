package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type IdentityManagerStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewIdentityManagerStub(client zbus.Client) *IdentityManagerStub {
	return &IdentityManagerStub{
		client: client,
		module: "identityd",
		object: zbus.ObjectID{
			Name:    "manager",
			Version: "0.0.1",
		},
	}
}

func (s *IdentityManagerStub) Decrypt(arg0 []uint8) (ret0 []uint8, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Decrypt", args...)
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

func (s *IdentityManagerStub) Encrypt(arg0 []uint8) (ret0 []uint8, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Encrypt", args...)
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

func (s *IdentityManagerStub) FarmID() (ret0 pkg.FarmID, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "FarmID", args...)
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

func (s *IdentityManagerStub) NodeID() (ret0 pkg.StrIdentifier) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "NodeID", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *IdentityManagerStub) PrivateKey() (ret0 []uint8) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "PrivateKey", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *IdentityManagerStub) Sign(arg0 []uint8) (ret0 []uint8, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Sign", args...)
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

func (s *IdentityManagerStub) Verify(arg0 []uint8, arg1 []uint8) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Verify", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
