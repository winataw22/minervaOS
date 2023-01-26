// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
)

type ProvisionStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewProvisionStub(client zbus.Client) *ProvisionStub {
	return &ProvisionStub{
		client: client,
		module: "provision",
		object: zbus.ObjectID{
			Name:    "provision",
			Version: "0.0.1",
		},
	}
}

func (s *ProvisionStub) DecommissionCached(ctx context.Context, arg0 string, arg1 string) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "DecommissionCached", args...)
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
