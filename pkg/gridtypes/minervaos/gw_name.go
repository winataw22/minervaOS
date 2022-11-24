package zos

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// GatewayNameProxy definition. this will proxy name.<zos.domain> to backends
type GatewayNameProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string `json:"name"`

	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool `json:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `json:"backends"`
}

func (g GatewayNameProxy) Valid(getter gridtypes.WorkloadGetter) error {
	if !gwNameRegex.MatchString(g.Name) {
		return fmt.Errorf("invalid name")
	}
	if len(g.Backends) == 0 {
		return fmt.Errorf("backends list can not be empty")
	}
	for _, backend := range g.Backends {
		if err := backend.Valid(g.TLSPassthrough); err != nil {
			return errors.Wrapf(err, "failed to validate backend '%s'", backend)
		}
	}

	return nil
}

func (g GatewayNameProxy) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", g.Name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", g.TLSPassthrough); err != nil {
		return err
	}

	for _, backend := range g.Backends {
		if _, err := fmt.Fprintf(w, "%s", string(backend)); err != nil {
			return err
		}
	}

	return nil
}

func (g GatewayNameProxy) Capacity() (gridtypes.Capacity, error) {
	// this has to be calculated per bytes served over the gw. so
	// a special handler in reporting that need to calculate and report
	// this.
	return gridtypes.Capacity{}, nil
}

// GatewayProxyResult results
type GatewayFQDNResult struct {
}
