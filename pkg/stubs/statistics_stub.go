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

func (s *StatisticsStub) Current(ctx context.Context) (ret0 gridtypes.Capacity) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Current", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *StatisticsStub) ReservedStream(ctx context.Context) (<-chan gridtypes.Capacity, error) {
	ch := make(chan gridtypes.Capacity)
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
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
