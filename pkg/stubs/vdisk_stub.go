package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	gridtypes "github.com/threefoldtech/zos/pkg/gridtypes"
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

func (s *VDiskModuleStub) Allocate(ctx context.Context, arg0 string, arg1 gridtypes.Unit) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Allocate", args...)
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

func (s *VDiskModuleStub) Deallocate(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Deallocate", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VDiskModuleStub) EnsureFilesystem(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "EnsureFilesystem", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VDiskModuleStub) Exists(ctx context.Context, arg0 string) (ret0 bool) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Exists", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VDiskModuleStub) Inspect(ctx context.Context, arg0 string) (ret0 pkg.VDisk, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Inspect", args...)
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

func (s *VDiskModuleStub) List(ctx context.Context) (ret0 []pkg.VDisk, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "List", args...)
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

func (s *VDiskModuleStub) WriteImage(ctx context.Context, arg0 string, arg1 string) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "WriteImage", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
