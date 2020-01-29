package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type VDiskModuleStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewVDiskModuleStub(client zbus.Client) *VDiskModuleStub {
	return &VDiskModuleStub{
		client: client,
		module: "storage",
		object: zbus.ObjectID{
			Name:    "vdisk",
			Version: "0.0.1",
		},
	}
}

func (s *VDiskModuleStub) Allocate(arg0 string, arg1 int64) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Allocate", args...)
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

func (s *VDiskModuleStub) Deallocate(arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Deallocate", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VDiskModuleStub) Exists(arg0 string) (ret0 bool) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Exists", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VDiskModuleStub) Inspect(arg0 string) (ret0 pkg.VDisk, ret1 error) {
	args := []interface{}{arg0}
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
