// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	gridtypes "github.com/threefoldtech/zos/pkg/gridtypes"
	zos "github.com/threefoldtech/zos/pkg/gridtypes/zos"
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

func (s *NetworkerStub) Addrs(ctx context.Context, arg0 string, arg1 string) (ret0 [][]uint8, ret1 string, ret2 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Addrs", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret2 = result.CallError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) CreateNR(ctx context.Context, arg0 gridtypes.WorkloadID, arg1 pkg.Network) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateNR", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) DMZAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses, 1)
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

func (s *NetworkerStub) DeleteNR(ctx context.Context, arg0 gridtypes.WorkloadID) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "DeleteNR", args...)
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

func (s *NetworkerStub) DisconnectPubTap(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "DisconnectPubTap", args...)
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

func (s *NetworkerStub) GetDefaultGwIP(ctx context.Context, arg0 zos.NetID) (ret0 []uint8, ret1 []uint8, ret2 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetDefaultGwIP", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret2 = result.CallError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetIPv6From4(ctx context.Context, arg0 zos.NetID, arg1 []uint8) (ret0 net.IPNet, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetIPv6From4", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetNet(ctx context.Context, arg0 zos.NetID) (ret0 net.IPNet, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNet", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetPublicConfig(ctx context.Context) (ret0 pkg.PublicConfig, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPublicConfig", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetPublicExitDevice(ctx context.Context) (ret0 pkg.ExitDevice, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPublicExitDevice", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetPublicIPv6Subnet(ctx context.Context) (ret0 net.IPNet, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPublicIPv6Subnet", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) GetSubnet(ctx context.Context, arg0 zos.NetID) (ret0 net.IPNet, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetSubnet", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) Interfaces(ctx context.Context, arg0 string, arg1 string) (ret0 interface{}, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Interfaces", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) Metrics(ctx context.Context) (ret0 pkg.NetResourceMetrics, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Metrics", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) Namespace(ctx context.Context, arg0 zos.NetID) (ret0 string) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Namespace", args...)
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

func (s *NetworkerStub) PubIPFilterExists(ctx context.Context, arg0 string) (ret0 bool) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "PubIPFilterExists", args...)
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

func (s *NetworkerStub) PubTapExists(ctx context.Context, arg0 string) (ret0 bool, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "PubTapExists", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) PublicAddresses(ctx context.Context) (<-chan pkg.OptionPublicConfig, error) {
	ch := make(chan pkg.OptionPublicConfig, 1)
	recv, err := s.client.Stream(ctx, s.module, s.object, "PublicAddresses")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.OptionPublicConfig
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

func (s *NetworkerStub) PublicIPv4Support(ctx context.Context) (ret0 bool) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "PublicIPv4Support", args...)
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

func (s *NetworkerStub) QSFSDestroy(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "QSFSDestroy", args...)
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

func (s *NetworkerStub) QSFSNamespace(ctx context.Context, arg0 string) (ret0 string) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "QSFSNamespace", args...)
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

func (s *NetworkerStub) QSFSPrepare(ctx context.Context, arg0 string) (ret0 string, ret1 string, ret2 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "QSFSPrepare", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret2 = result.CallError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) QSFSYggIP(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "QSFSYggIP", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) Ready(ctx context.Context) (ret0 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Ready", args...)
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

func (s *NetworkerStub) RemovePubIPFilter(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "RemovePubIPFilter", args...)
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

func (s *NetworkerStub) RemovePubTap(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "RemovePubTap", args...)
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

func (s *NetworkerStub) RemoveTap(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "RemoveTap", args...)
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

func (s *NetworkerStub) SetPublicConfig(ctx context.Context, arg0 pkg.PublicConfig) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetPublicConfig", args...)
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

func (s *NetworkerStub) SetPublicExitDevice(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetPublicExitDevice", args...)
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

func (s *NetworkerStub) SetupMyceliumTap(ctx context.Context, arg0 string, arg1 zos.NetID, arg2 zos.MyceliumIP) (ret0 pkg.PlanetaryTap, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetupMyceliumTap", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) SetupPrivTap(ctx context.Context, arg0 zos.NetID, arg1 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetupPrivTap", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) SetupPubIPFilter(ctx context.Context, arg0 string, arg1 string, arg2 []uint8, arg3 []uint8, arg4 string) (ret0 error) {
	args := []interface{}{arg0, arg1, arg2, arg3, arg4}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetupPubIPFilter", args...)
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

func (s *NetworkerStub) SetupPubTap(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetupPubTap", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) SetupYggTap(ctx context.Context, arg0 string) (ret0 pkg.PlanetaryTap, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetupYggTap", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) TapExists(ctx context.Context, arg0 string) (ret0 bool, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "TapExists", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) UnsetPublicConfig(ctx context.Context) (ret0 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UnsetPublicConfig", args...)
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

func (s *NetworkerStub) WireguardPorts(ctx context.Context) (ret0 []uint, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "WireguardPorts", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) YggAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses, 1)
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

func (s *NetworkerStub) ZDBDestroy(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "ZDBDestroy", args...)
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

func (s *NetworkerStub) ZDBPrepare(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "ZDBPrepare", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *NetworkerStub) ZOSAddresses(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses, 1)
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
