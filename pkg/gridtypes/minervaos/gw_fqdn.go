package zos

import (
	"fmt"
	"io"
	"regexp"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	gwNameRegex = regexp.MustCompile(`^[a-zA-Z0-9-.]+$`)
)

// GatewayFQDNProxy definition. this will proxy name.<zos.domain> to backends
type GatewayFQDNProxy struct {
	GatewayBase

	// FQDN the fully qualified domain name to use (cannot be present with Name)
	FQDN string `json:"fqdn"`
}

func (g GatewayFQDNProxy) Valid(getter gridtypes.WorkloadGetter) error {
	if !gwNameRegex.MatchString(g.FQDN) {
		return fmt.Errorf("invalid name")
	}
	if g.FQDN[len(g.FQDN)-1] == '.' {
		return fmt.Errorf("fqdn can't end with a dot")
	}

	return g.GatewayBase.Valid(getter)
}

func (g GatewayFQDNProxy) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", g.FQDN); err != nil {
		return err
	}

	return g.GatewayBase.Challenge(w)
}

func (g GatewayFQDNProxy) Capacity() (gridtypes.Capacity, error) {
	// this has to be calculated per bytes served over the gw. so
	// a special handler in reporting that need to calculate and report
	// this.
	return gridtypes.Capacity{}, nil
}

// GatewayProxyResult results
type GatewayFQDNResult struct {
}
