// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type SystemMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewSystemMonitorStub(client zbus.Client) *SystemMonitorStub {
	return &SystemMonitorStub{
		client: client,
		module: "node",
		object: zbus.ObjectID{
			Name:    "system",
			Version: "0.0.1",
		},
	}
}

func (s *SystemMonitorStub) CPU(ctx context.Context) (<-chan pkg.TimesStat, error) {
	ch := make(chan pkg.TimesStat, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "CPU")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.TimesStat
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

func (s *SystemMonitorStub) Disks(ctx context.Context) (<-chan pkg.DisksIOCountersStat, error) {
	ch := make(chan pkg.DisksIOCountersStat, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Disks")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.DisksIOCountersStat
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

func (s *SystemMonitorStub) GetNodeFeatures(ctx context.Context) (ret0 []pkg.NodeFeature) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeFeatures", args...)
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

func (s *SystemMonitorStub) Memory(ctx context.Context) (<-chan pkg.VirtualMemoryStat, error) {
	ch := make(chan pkg.VirtualMemoryStat, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Memory")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.VirtualMemoryStat
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

func (s *SystemMonitorStub) Nics(ctx context.Context) (<-chan pkg.NicsIOCounterStat, error) {
	ch := make(chan pkg.NicsIOCounterStat, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Nics")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NicsIOCounterStat
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

func (s *SystemMonitorStub) NodeID(ctx context.Context) (ret0 uint32) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "NodeID", args...)
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
