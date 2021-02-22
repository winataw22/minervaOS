package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	zos "github.com/threefoldtech/zos/pkg/gridtypes/zos"
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

func (s *StorageModuleStub) Allocate(arg0 string, arg1 zos.DeviceType, arg2 uint64, arg3 zos.ZDBMode) (ret0 pkg.Allocation, ret1 error) {
	args := []interface{}{arg0, arg1, arg2, arg3}
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

func (s *StorageModuleStub) BrokenDevices() (ret0 []pkg.BrokenDevice) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "BrokenDevices", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *StorageModuleStub) BrokenPools() (ret0 []pkg.BrokenPool) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "BrokenPools", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *StorageModuleStub) CreateFilesystem(arg0 string, arg1 uint64, arg2 zos.DeviceType) (ret0 pkg.Filesystem, ret1 error) {
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

func (s *StorageModuleStub) Find(arg0 string) (ret0 pkg.Allocation, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Find", args...)
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

func (s *StorageModuleStub) GetCacheFS() (ret0 pkg.Filesystem, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "GetCacheFS", args...)
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

func (s *StorageModuleStub) ListFilesystems() (ret0 []pkg.Filesystem, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "ListFilesystems", args...)
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

func (s *StorageModuleStub) Monitor(ctx context.Context) (<-chan pkg.PoolsStats, error) {
	ch := make(chan pkg.PoolsStats)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Monitor")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.PoolsStats
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *StorageModuleStub) Path(arg0 string) (ret0 pkg.Filesystem, ret1 error) {
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

func (s *StorageModuleStub) Total(arg0 zos.DeviceType) (ret0 uint64, ret1 error) {
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
