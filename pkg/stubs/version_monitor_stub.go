package stubs

import (
	"context"
	semver "github.com/blang/semver"
	zbus "github.com/threefoldtech/zbus"
)

type VersionMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewVersionMonitorStub(client zbus.Client) *VersionMonitorStub {
	return &VersionMonitorStub{
		client: client,
		module: "identityd",
		object: zbus.ObjectID{
			Name:    "monitor",
			Version: "0.0.1",
		},
	}
}

func (s *VersionMonitorStub) GetVersion(ctx context.Context) (ret0 semver.Version) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetVersion", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *VersionMonitorStub) Version(ctx context.Context) (<-chan semver.Version, error) {
	ch := make(chan semver.Version)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Version")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj semver.Version
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
