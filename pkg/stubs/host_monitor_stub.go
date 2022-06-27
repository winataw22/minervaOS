package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	"time"
)

type HostMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewHostMonitorStub(client zbus.Client) *HostMonitorStub {
	return &HostMonitorStub{
		client: client,
		module: "node",
		object: zbus.ObjectID{
			Name:    "host",
			Version: "0.0.1",
		},
	}
}

func (s *HostMonitorStub) Uptime(ctx context.Context) (<-chan time.Duration, error) {
	ch := make(chan time.Duration, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Uptime")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj time.Duration
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
