// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type PerformanceMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewPerformanceMonitorStub(client zbus.Client) *PerformanceMonitorStub {
	return &PerformanceMonitorStub{
		client: client,
		module: "node",
		object: zbus.ObjectID{
			Name:    "performance-monitor",
			Version: "0.0.1",
		},
	}
}

func (s *PerformanceMonitorStub) Get(ctx context.Context, arg0 string) (ret0 pkg.TaskResult, ret1 error) {
	args := []interface{}{arg0}
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

func (s *PerformanceMonitorStub) GetAll(ctx context.Context) (ret0 []pkg.TaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetAll", args...)
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
