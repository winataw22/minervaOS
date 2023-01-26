// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type GatewayStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewGatewayStub(client zbus.Client) *GatewayStub {
	return &GatewayStub{
		client: client,
		module: "gateway",
		object: zbus.ObjectID{
			Name:    "manager",
			Version: "0.0.1",
		},
	}
}

func (s *GatewayStub) DeleteNamedProxy(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "DeleteNamedProxy", args...)
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

func (s *GatewayStub) Metrics(ctx context.Context) (ret0 pkg.GatewayMetrics, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Metrics", args...)
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

func (s *GatewayStub) SetFQDNProxy(ctx context.Context, arg0 string, arg1 string, arg2 []string, arg3 bool) (ret0 error) {
	args := []interface{}{arg0, arg1, arg2, arg3}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetFQDNProxy", args...)
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

func (s *GatewayStub) SetNamedProxy(ctx context.Context, arg0 string, arg1 string, arg2 []string, arg3 bool, arg4 uint32) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1, arg2, arg3, arg4}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetNamedProxy", args...)
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
