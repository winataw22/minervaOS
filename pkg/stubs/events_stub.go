package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type EventsStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewEventsStub(client zbus.Client) *EventsStub {
	return &EventsStub{
		client: client,
		module: "node",
		object: zbus.ObjectID{
			Name:    "events",
			Version: "0.0.1",
		},
	}
}

func (s *EventsStub) ContractCancelledEvent(ctx context.Context) (<-chan pkg.ContractCancelledEvent, error) {
	ch := make(chan pkg.ContractCancelledEvent)
	recv, err := s.client.Stream(ctx, s.module, s.object, "ContractCancelledEvent")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.ContractCancelledEvent
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

func (s *EventsStub) ContractLockedEvent(ctx context.Context) (<-chan pkg.ContractLockedEvent, error) {
	ch := make(chan pkg.ContractLockedEvent)
	recv, err := s.client.Stream(ctx, s.module, s.object, "ContractLockedEvent")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.ContractLockedEvent
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

func (s *EventsStub) PublicConfigEvent(ctx context.Context) (<-chan pkg.PublicConfigEvent, error) {
	ch := make(chan pkg.PublicConfigEvent)
	recv, err := s.client.Stream(ctx, s.module, s.object, "PublicConfigEvent")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.PublicConfigEvent
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
