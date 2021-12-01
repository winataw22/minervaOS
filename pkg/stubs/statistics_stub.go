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

func (s *StatisticsStub) Reserved(ctx context.Context) (<-chan gridtypes.Capacity, error) {
	ch := make(chan gridtypes.Capacity)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Reserved")
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
