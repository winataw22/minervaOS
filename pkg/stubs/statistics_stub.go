// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	gridtypes "github.com/threefoldtech/zos/pkg/gridtypes"
)

type StatisticsStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewStatisticsStub(client zbus.Client) *StatisticsStub {
	return &StatisticsStub{
		client: client,
		module: "provision",
		object: zbus.ObjectID{
			Name:    "statistics",
			Version: "0.0.1",
		},
	}
}

func (s *StatisticsStub) Current(ctx context.Context) (ret0 gridtypes.Capacity, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Current", args...)
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

func (s *StatisticsStub) ReservedStream(ctx context.Context) (<-chan gridtypes.Capacity, error) {
	ch := make(chan gridtypes.Capacity, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "ReservedStream")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj gridtypes.Capacity
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			select {
			case <-ctx.Done():
				return
			case ch <- obj:
			default:
			}
		}
	}()
	return ch, nil
}

func (s *StatisticsStub) Total(ctx context.Context) (ret0 gridtypes.Capacity) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Total", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *StatisticsStub) Workloads(ctx context.Context) (ret0 int, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Workloads", args...)
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
