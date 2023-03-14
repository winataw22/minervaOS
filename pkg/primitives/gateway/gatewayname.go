package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*NameManager)(nil)
)

type NameManager struct {
	zbus zbus.Client
}

func NewNameManager(zbus zbus.Client) *NameManager {
	return &NameManager{zbus}
}

func (p *NameManager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {

	result := zos.GatewayProxyResult{}
	var proxy zos.GatewayNameProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}

	gateway := stubs.NewGatewayStub(p.zbus)
	fqdn, err := gateway.SetNamedProxy(ctx, wl.ID.String(), proxy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup name proxy")
	}
	result.FQDN = fqdn
	log.Debug().Str("domain", fqdn).Msg("domain reserved")
	return result, nil
}

func (p *NameManager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.DeleteNamedProxy(ctx, wl.ID.String()); err != nil {
		return errors.Wrap(err, "failed to delete name proxy")
	}
	return nil
}
