// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	gridtypes "github.com/threefoldtech/zos/pkg/gridtypes"
)

type ProvisionStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewProvisionStub(client zbus.Client) *ProvisionStub {
	return &ProvisionStub{
		client: client,
		module: "provision",
		object: zbus.ObjectID{
			Name:    "provision",
			Version: "0.0.1",
		},
	}
}

func (s *ProvisionStub) Changes(ctx context.Context, arg0 uint32, arg1 uint64) (ret0 []gridtypes.Workload, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Changes", args...)
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

func (s *ProvisionStub) CreateOrUpdate(ctx context.Context, arg0 uint32, arg1 gridtypes.Deployment, arg2 bool) (ret0 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateOrUpdate", args...)
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

func (s *ProvisionStub) DecommissionCached(ctx context.Context, arg0 string, arg1 string) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "DecommissionCached", args...)
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

func (s *ProvisionStub) Get(ctx context.Context, arg0 uint32, arg1 uint64) (ret0 gridtypes.Deployment, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Get", args...)
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

func (s *ProvisionStub) GetWorkloadStatus(ctx context.Context, arg0 string) (ret0 gridtypes.ResultState, ret1 bool, ret2 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetWorkloadStatus", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret2 = result.CallError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *ProvisionStub) List(ctx context.Context, arg0 uint32) (ret0 []gridtypes.Deployment, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "List", args...)
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

func (s *ProvisionStub) ListPrivateIPs(ctx context.Context, arg0 uint32, arg1 gridtypes.Name) (ret0 []string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "ListPrivateIPs", args...)
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

func (s *ProvisionStub) ListPublicIPs(ctx context.Context) (ret0 []string, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "ListPublicIPs", args...)
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
