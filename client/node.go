package client

import (
	"context"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type NodeClient struct {
	nodeTwin uint32
	bus      rmb.Client
}

type args map[string]interface{}

func NewNodeClient(nodeTwin uint32, bus rmb.Client) *NodeClient {
	return &NodeClient{nodeTwin, bus}
}

func (n *NodeClient) DeploymentDeploy(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.deploy"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

func (n *NodeClient) DeploymentUpdate(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.update"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

func (n *NodeClient) DeploymentGet(ctx context.Context, deploymentID uint32) (dl gridtypes.Deployment, err error) {
	const cmd = "zos.deployment.get"
	in := args{
		"deployment_id": deploymentID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &dl); err != nil {
		return dl, err
	}

	return dl, nil
}

func (n *NodeClient) DeploymentDelete(ctx context.Context, deploymentID uint32) error {
	const cmd = "zos.deployment.delete"
	in := args{
		"deployment_id": deploymentID,
	}

	return n.bus.Call(ctx, n.nodeTwin, cmd, in, nil)
}

func (n *NodeClient) Counters(ctx context.Context) (total gridtypes.Capacity, used gridtypes.Capacity, err error) {
	const cmd = "zos.statistics.get"
	var result struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}
	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return result.Total, result.Used, nil
}

func (n *NodeClient) NetworkListWGPorts(ctx context.Context) ([]uint16, error) {
	const cmd = "zos.network.list_wg_ports"
	var result []uint16

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *NodeClient) NetworkListIPs(ctx context.Context) ([]string, error) {
	const cmd = "zos.network.list_public_ips"
	var result []string

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *NodeClient) NetworkGetPublicConfig(ctx context.Context) (cfg pkg.PublicConfig, err error) {
	const cmd = "zos.network.public_config_get"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &cfg); err != nil {
		return
	}

	return
}
