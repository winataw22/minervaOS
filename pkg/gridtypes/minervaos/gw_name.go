package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// GatewayNameProxy definition. this will proxy name.<zos.domain> to backends
type GatewayNameProxy struct {
	GatewayBase
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string `json:"name"`
}

func (g GatewayNameProxy) Valid(getter gridtypes.WorkloadGetter) error {
	if !gwNameRegex.MatchString(g.Name) {
		return fmt.Errorf("invalid name")
	}

	return g.GatewayBase.Valid(getter)
}

func (g GatewayNameProxy) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", g.Name); err != nil {
		return err
	}

	return g.GatewayBase.Challenge(w)
}

func (g GatewayNameProxy) Capacity() (gridtypes.Capacity, error) {
	// this has to be calculated per bytes served over the gw. so
	// a special handler in reporting that need to calculate and report
	// this.
	return gridtypes.Capacity{}, nil
}

// GatewayProxyResult results
type GatewayProxyResult struct {
	FQDN string `json:"fqdn"`
}
