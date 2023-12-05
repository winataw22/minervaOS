// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
)

type ZUIStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewZUIStub(client zbus.Client) *ZUIStub {
	return &ZUIStub{
		client: client,
		module: "zui",
		object: zbus.ObjectID{
			Name:    "zui",
			Version: "0.0.1",
		},
	}
}

func (s *ZUIStub) PushErrors(ctx context.Context, arg0 string, arg1 []string) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "PushErrors", args...)
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
