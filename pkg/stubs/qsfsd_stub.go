// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	zos "github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type QSFSDStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewQSFSDStub(client zbus.Client) *QSFSDStub {
	return &QSFSDStub{
		client: client,
		module: "qsfsd",
		object: zbus.ObjectID{
			Name:    "manager",
			Version: "0.0.1",
		},
	}
}

func (s *QSFSDStub) Metrics(ctx context.Context) (ret0 pkg.QSFSMetrics, ret1 error) {
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

func (s *QSFSDStub) Mount(ctx context.Context, arg0 string, arg1 zos.QuantumSafeFS) (ret0 pkg.QSFSInfo, ret1 error) {
	args := []interface{}{arg0, arg1}
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

func (s *QSFSDStub) SignalDelete(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SignalDelete", args...)
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

func (s *QSFSDStub) UpdateMount(ctx context.Context, arg0 string, arg1 zos.QuantumSafeFS) (ret0 pkg.QSFSInfo, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateMount", args...)
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
