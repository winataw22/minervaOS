package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	"net"
)

type NetworkerStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewNetworkerStub(client zbus.Client) *NetworkerStub {
	return &NetworkerStub{
		client: client,
		module: "network",
		object: zbus.ObjectID{
			Name:    "network",
			Version: "0.0.1",
		},
	}
}

func (s *NetworkerStub) Addrs(arg0 string, arg1 string) (ret0 [][]uint8, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Addrs", args...)
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

func (s *NetworkerStub) CreateNR(arg0 pkg.NetResource) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "CreateNR", args...)
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

func (s *NetworkerStub) DMZAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses)
	recv, err := s.client.Stream(ctx, s.module, s.object, "DMZAddresses")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NetlinkAddresses
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *NetworkerStub) DeleteNR(arg0 pkg.NetResource) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "DeleteNR", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetDefaultGwIP(arg0 pkg.NetID) (ret0 []uint8, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "GetDefaultGwIP", args...)
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

func (s *NetworkerStub) GetSubnet(arg0 pkg.NetID) (ret0 net.IPNet, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "GetSubnet", args...)
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

func (s *NetworkerStub) Join(arg0 pkg.NetID, arg1 string, arg2 pkg.ContainerNetworkConfig) (ret0 pkg.Member, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.Request(s.module, s.object, "Join", args...)
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

func (s *NetworkerStub) Leave(arg0 pkg.NetID, arg1 string) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Leave", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) PublicAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses)
	recv, err := s.client.Stream(ctx, s.module, s.object, "PublicAddresses")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NetlinkAddresses
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *NetworkerStub) Ready() (ret0 error) {
	args := []interface{}{}
	result, err := s.client.Request(s.module, s.object, "Ready", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) RemoveTap(arg0 pkg.NetID) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "RemoveTap", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) SetupTap(arg0 pkg.NetID) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "SetupTap", args...)
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

func (s *NetworkerStub) TapExists(arg0 pkg.NetID) (ret0 bool, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "TapExists", args...)
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

func (s *NetworkerStub) YggAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses)
	recv, err := s.client.Stream(ctx, s.module, s.object, "YggAddresses")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NetlinkAddresses
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *NetworkerStub) ZDBDestroy(arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "ZDBDestroy", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) ZDBPrepare(arg0 []uint8) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.Request(s.module, s.object, "ZDBPrepare", args...)
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

func (s *NetworkerStub) ZOSAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses)
	recv, err := s.client.Stream(ctx, s.module, s.object, "ZOSAddresses")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NetlinkAddresses
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
