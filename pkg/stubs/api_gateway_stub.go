// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	tfchainclientgo "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type SubstrateGatewayStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewSubstrateGatewayStub(client zbus.Client) *SubstrateGatewayStub {
	return &SubstrateGatewayStub{
		client: client,
		module: "api-gateway",
		object: zbus.ObjectID{
			Name:    "api-gateway",
			Version: "0.0.1",
		},
	}
}

func (s *SubstrateGatewayStub) CreateNode(ctx context.Context, arg0 tfchainclientgo.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateNode", args...)
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

func (s *SubstrateGatewayStub) CreateTwin(ctx context.Context, arg0 string, arg1 []uint8) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateTwin", args...)
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

func (s *SubstrateGatewayStub) EnsureAccount(ctx context.Context, arg0 []string, arg1 string, arg2 string) (ret0 tfchainclientgo.AccountInfo, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "EnsureAccount", args...)
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

func (s *SubstrateGatewayStub) GetContract(ctx context.Context, arg0 uint64) (ret0 tfchainclientgo.Contract, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *SubstrateGatewayStub) GetContractIDByNameRegistration(ctx context.Context, arg0 string) (ret0 uint64, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContractIDByNameRegistration", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *SubstrateGatewayStub) GetFarm(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.Farm, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetFarm", args...)
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

func (s *SubstrateGatewayStub) GetNode(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.Node, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNode", args...)
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

func (s *SubstrateGatewayStub) GetNodeByTwinID(ctx context.Context, arg0 uint32) (ret0 uint32, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeByTwinID", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *SubstrateGatewayStub) GetNodeContracts(ctx context.Context, arg0 uint32) (ret0 []types.U64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeContracts", args...)
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

func (s *SubstrateGatewayStub) GetNodeRentContract(ctx context.Context, arg0 uint32) (ret0 uint64, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeRentContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *SubstrateGatewayStub) GetNodes(ctx context.Context, arg0 uint32) (ret0 []uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodes", args...)
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

func (s *SubstrateGatewayStub) GetPowerTarget(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.NodePower, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPowerTarget", args...)
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

func (s *SubstrateGatewayStub) GetTwin(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.Twin, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwin", args...)
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

func (s *SubstrateGatewayStub) GetTwinByPubKey(ctx context.Context, arg0 []uint8) (ret0 uint32, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwinByPubKey", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *SubstrateGatewayStub) Report(ctx context.Context, arg0 []tfchainclientgo.NruConsumption) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Report", args...)
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

func (s *SubstrateGatewayStub) SetContractConsumption(ctx context.Context, arg0 ...tfchainclientgo.ContractResources) (ret0 error) {
	args := []interface{}{}
	for _, argv := range arg0 {
		args = append(args, argv)
	}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetContractConsumption", args...)
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

func (s *SubstrateGatewayStub) SetNodePowerState(ctx context.Context, arg0 bool) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetNodePowerState", args...)
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

func (s *SubstrateGatewayStub) UpdateNode(ctx context.Context, arg0 tfchainclientgo.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNode", args...)
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

func (s *SubstrateGatewayStub) UpdateNodeUptimeV2(ctx context.Context, arg0 uint64, arg1 uint64) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNodeUptimeV2", args...)
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
