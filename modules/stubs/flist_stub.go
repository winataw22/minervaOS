package stubs

import zbus "github.com/threefoldtech/zbus"

type FlisterStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewFlisterStub(client zbus.Client) *FlisterStub {
	return &FlisterStub{
		client: client,
		module: "flist",
		object: zbus.ObjectID{
			Name:    "flist",
			Version: "0.0.1",
		},
	}
}

func (s *FlisterStub) Mount(arg0 string, arg1 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Mount", args...)
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


func (s *FlisterStub) Umount(arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "Umount", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}
