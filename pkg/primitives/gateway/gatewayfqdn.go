package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*FQDNManager)(nil)
)

type FQDNManager struct {
	zbus zbus.Client
}

func NewFQDNManager(zbus zbus.Client) *FQDNManager {
	return &FQDNManager{zbus}
}

func (p *FQDNManager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	result := zos.GatewayFQDNResult{}
	var proxy zos.GatewayFQDNProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}
	backends := make([]string, len(proxy.Backends))
	for idx, backend := range proxy.Backends {
		backends[idx] = string(backend)
	}
	gateway := stubs.NewGatewayStub(p.zbus)
	err := gateway.SetFQDNProxy(ctx, wl.ID.String(), proxy.FQDN, backends, proxy.TLSPassthrough)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup fqdn proxy")
	}
	return result, nil
}

func (p *FQDNManager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.DeleteNamedProxy(ctx, wl.ID.String()); err != nil {
		return errors.Wrap(err, "failed to delete fqdn proxy")
	}
	return nil
}
