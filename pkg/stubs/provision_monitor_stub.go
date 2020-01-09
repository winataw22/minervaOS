package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type ProvisionMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewProvisionMonitorStub(client zbus.Client) *ProvisionMonitorStub {
	return &ProvisionMonitorStub{
		client: client,
		module: "provision",
		object: zbus.ObjectID{
			Name:    "provision",
			Version: "0.0.1",
		},
	}
}

func (s *ProvisionMonitorStub) Counters(ctx context.Context) (<-chan pkg.ProvisionCounters, error) {
	ch := make(chan pkg.ProvisionCounters)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Counters")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.ProvisionCounters
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
