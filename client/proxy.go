package client

import (
	"context"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type NodeStatus struct {
	Current    gridtypes.Capacity `json:"used"`
	Total      gridtypes.Capacity `json:"total"`
	ZosVersion string             `json:"zos"`
	Hypervisor string             `json:"hypervisor"`
}

// ProxyClient struct
type ProxyClient struct {
	twin uint32
	bus  rmb.Client
}

// NewNodeClient creates a new node RMB client. This client then can be used to
// communicate with the node over RMB.
func NewProxyClient(twin uint32, bus rmb.Client) *ProxyClient {
	return &ProxyClient{twin, bus}
}

//
func (n *ProxyClient) ReportStats(ctx context.Context, report NodeStatus) error {
	const cmd = "proxy.status.report"
	return n.bus.Call(ctx, n.twin, cmd, report, nil)
}
