package zosapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (g *ZosAPI) deploymentDeployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := json.Unmarshal(payload, &deployment); err != nil {
		return nil, err
	}
	err := g.provisionStub.CreateOrUpdate(ctx, peer.GetTwinID(ctx), deployment, false)
	return nil, err
}

func (g *ZosAPI) deploymentUpdateHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := json.Unmarshal(payload, &deployment); err != nil {
		return nil, err
	}
	err := g.provisionStub.CreateOrUpdate(ctx, peer.GetTwinID(ctx), deployment, true)
	return nil, err
}

func (g *ZosAPI) deploymentDeleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return nil, fmt.Errorf("deletion over the api is disabled, please cancel your contract instead")
}

func (g *ZosAPI) deploymentGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		ContractID uint64 `json:"contract_id"`
	}
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, err
	}

	return g.provisionStub.Get(ctx, peer.GetTwinID(ctx), args.ContractID)

}

func (g *ZosAPI) deploymentListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.provisionStub.List(ctx, peer.GetTwinID(ctx))
}

func (g *ZosAPI) deploymentChangesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		ContractID uint64 `json:"contract_id"`
	}
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, err
	}
	return g.provisionStub.Changes(ctx, peer.GetTwinID(ctx), args.ContractID)
}
