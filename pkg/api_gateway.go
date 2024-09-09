package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

//go:generate zbusc -module api-gateway -version 0.0.1 -name api-gateway -package stubs github.com/threefoldtech/zos/pkg+SubstrateGateway stubs/api_gateway_stub.go

type SubstrateGateway interface {
	CreateNode(node substrate.Node) (uint32, error)
	CreateTwin(relay string, pk []byte) (uint32, error)
	EnsureAccount(activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error)
	GetContract(id uint64) (substrate.Contract, SubstrateError)
	GetContractIDByNameRegistration(name string) (uint64, SubstrateError)
	GetFarm(id uint32) (substrate.Farm, error)
	GetNode(id uint32) (substrate.Node, error)
	GetNodeByTwinID(twin uint32) (uint32, SubstrateError)
	GetNodeContracts(node uint32) ([]types.U64, error)
	GetNodeRentContract(node uint32) (uint64, SubstrateError)
	GetNodes(farmID uint32) ([]uint32, error)
	GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error)
	GetTwin(id uint32) (substrate.Twin, error)
	GetTwinByPubKey(pk []byte) (uint32, SubstrateError)
	Report(consumptions []substrate.NruConsumption) (types.Hash, error)
	SetContractConsumption(resources ...substrate.ContractResources) error
	SetNodePowerState(up bool) (hash types.Hash, err error)
	UpdateNode(node substrate.Node) (uint32, error)
	UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error)
}

type SubstrateError struct {
	Err  error
	Code int
}

func (e *SubstrateError) IsError() bool {
	return e.Code != CodeNoError
}

func (e *SubstrateError) IsCode(codes ...int) bool {
	for _, code := range codes {
		if code == e.Code {
			return true
		}
	}
	return false
}

const (
	CodeGenericError = iota - 1
	CodeNoError
	CodeNotFound
	CodeBurnTransactionNotFound
	CodeRefundTransactionNotFound
	CodeFailedToDecode
	CodeInvalidVersion
	CodeUnknownVersion
	CodeIsUsurped
	CodeAccountNotFound
	CodeDepositFeeNotFound
	CodeMintTransactionNotFound
)
