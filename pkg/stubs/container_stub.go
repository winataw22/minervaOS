package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type ContainerModuleStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewContainerModuleStub(client zbus.Client) *ContainerModuleStub {
	return &ContainerModuleStub{
		client: client,
		module: "container",
		object: zbus.ObjectID{
			Name:    "container",
			Version: "0.0.1",
		},
	}
}

func (s *ContainerModuleStub) Delete(arg0 string, arg1 pkg.ContainerID) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Delete", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *ContainerModuleStub) Inspect(arg0 string, arg1 pkg.ContainerID) (ret0 pkg.Container, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Inspect", args...)
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

func (s *ContainerModuleStub) List(arg0 string) (ret0 []pkg.ContainerID, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "List", args...)
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

func (s *ContainerModuleStub) ListNS() (ret0 []string, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "ListNS", args...)
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

func (s *ContainerModuleStub) Run(arg0 string, arg1 pkg.Container) (ret0 pkg.ContainerID, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Run", args...)
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
