// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	gridtypes "github.com/threefoldtech/zos/pkg/gridtypes"
)

type FlisterStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewFlisterStub(client zbus.Client) *FlisterStub {
	return &FlisterStub{
		client: client,
		module: "flist",
		object: zbus.ObjectID{
			Name:    "flist",
			Version: "0.0.1",
		},
	}
}

func (s *FlisterStub) Exists(ctx context.Context, arg0 string) (ret0 bool, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Exists", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *FlisterStub) FlistHash(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "FlistHash", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *FlisterStub) HashFromRootPath(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "HashFromRootPath", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *FlisterStub) Mount(ctx context.Context, arg0 string, arg1 string, arg2 pkg.MountOptions) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Mount", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *FlisterStub) Unmount(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Unmount", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret0 = result.CallError()
	loader := zbus.Loader{}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *FlisterStub) UpdateMountSize(ctx context.Context, arg0 string, arg1 gridtypes.Unit) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateMountSize", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}
