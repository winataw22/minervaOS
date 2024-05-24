# API Gateway Module

## ZBus

API Gateway module is available on ZBus over the following channel:
| module | object | version |
|--------|--------|---------|
| api-gateway|[api-gateway](#interface)| 0.0.1|

## Introduction

API Gateway module acts as the entrypoint for incoming and outgoing requests. Modules trying to reach Threefold Chain should do that through API Gateway, that way we can ensure no extrinsic chain requests are done at the same time causing some of them to be ignored. Incoming RMB requests should also go through API Gateway and be passed to each module through internal communication (ZBus). Having all routes defined on one place rather than being scattered around in every module highly improves readability and also the ability to traverse different API implementation. Developer of each module that requires an external API needs to define the entrypoint (i.e. `zos.deployment.deploy`) and pass user input after validation to the module internal API.

## zinit unit

`api-gateway` module needs to start after `identityd` has started as it needs the node identity for managing chain requests.

```yaml
exec: api-gateway --broker unix:///var/run/redis.sock
after:
  - boot
  - identityd
```

## Interface

```go
type SubstrateGateway interface {
    CreateNode(node substrate.Node) (uint32, error)
    CreateTwin(relay string, pk []byte) (uint32, error)
    EnsureAccount(activationURL string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error)
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
```
