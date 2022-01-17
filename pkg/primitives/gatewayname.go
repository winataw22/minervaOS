package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func validateNameContract(twinID uint32, name string) error {
	// TODO: validate against substrate?
	return nil
}

func (p *Primitives) gwProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {

	result := zos.GatewayProxyResult{}
	var proxy zos.GatewayNameProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}
	backends := make([]string, len(proxy.Backends))
	for idx, backend := range proxy.Backends {
		backends[idx] = string(backend)
	}
	// what we need to do:
	// - does this node support gateways ?
	// this can be validated by checking if we have a "public" namespace
	twinID, _ := provision.GetDeploymentID(ctx)
	if err := validateNameContract(twinID, proxy.Name); err != nil {
		return nil, errors.Wrap(err, "failed to validate name contract")
	}
	// - Validation of ownership of the name (later)
	// this must be done against substrate. Make sure that same user (twin) owns the
	// name int he workload config

	// - make necessary calls to gateway daemon.
	// gateway := stubs.NewGatewayStub(p.zbus)
	// gateway.SetNamedProxy(ctx context.Context, arg0 string, arg1 []string)
	gateway := stubs.NewGatewayStub(p.zbus)
	fqdn, err := gateway.SetNamedProxy(ctx, wl.ID.String(), proxy.Name, backends, proxy.TLSPassthrough, twinID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup name proxy")
	}
	result.FQDN = fqdn
	log.Debug().Str("domain", fqdn).Msg("domain reserved")
	return result, nil
}

func (p *Primitives) gwDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.DeleteNamedProxy(ctx, wl.ID.String()); err != nil {
		return errors.Wrap(err, "failed to delete name proxy")
	}
	return nil
}
