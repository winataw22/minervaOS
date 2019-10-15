package stubs

import (
	semver "github.com/blang/semver"
	zbus "github.com/threefoldtech/zbus"
)

type UpgradeModuleStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewUpgradeModuleStub(client zbus.Client) *UpgradeModuleStub {
	return &UpgradeModuleStub{
		client: client,
		module: "upgrade",
		object: zbus.ObjectID{
			Name:    "upgrade",
			Version: "0.0.1",
		},
	}
}

func (s *UpgradeModuleStub) Version() (ret0 semver.Version, ret1 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "Version", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	ret1 = new(zbus.RemoteError)
	if err := result.Unmarshal(1, &ret1); err != nil {
		panic(err)
	}
	return
}
