package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type StorageModuleStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewStorageModuleStub(client zbus.Client) *StorageModuleStub {
	return &StorageModuleStub{
		client: client,
		module: "storage",
		object: zbus.ObjectID{
			Name:    "storage",
			Version: "0.0.1",
		},
	}
}

func (s *StorageModuleStub) Allocate(arg0 pkg.DeviceType, arg1 uint64, arg2 pkg.ZDBMode) (ret0 string, ret1 string, ret2 error) {
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

func (s *StorageModuleStub) Claim(arg0 string, arg1 uint64) (ret0 error) {
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

func (s *StorageModuleStub) CreateFilesystem(arg0 string, arg1 uint64, arg2 pkg.DeviceType) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.Request(s.module, s.object, "CreateFilesystem", args...)
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

func (s *StorageModuleStub) Path(arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Path", args...)
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

func (s *StorageModuleStub) ReleaseFilesystem(arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "ReleaseFilesystem", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *StorageModuleStub) Total(arg0 pkg.DeviceType) (ret0 uint64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Total", args...)
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
