package zos

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	gwNameRegex = regexp.MustCompile(`^[a-zA-Z0-9-.]+$`)
)

type Backend string

// check if valid http://x.x.x.x:port or [::]:port
func (b Backend) Valid() error {
	u, err := url.Parse(string(b))
	if err != nil {
		return errors.Wrap(err, "failed to parse backend")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid scheme expected http, or https")
	}

	ip := net.ParseIP(u.Hostname())
	if ip == nil || len(ip) == 0 || ip.IsLoopback() {
		return fmt.Errorf("invalid ip address in backend: %s", u.Hostname())
	}
	return nil
}

// GatewayFQDNProxy definition. this will proxy name.<zos.domain> to backends
type GatewayFQDNProxy struct {
	// FQDN the fully qualified domain name to use (cannot be present with Name)
	FQDN string `json:"fqdn"`

	// Passthroug whether to pass tls traffic or not
	TLSPassthrough bool `json:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `json:"backends"`
}

func (g GatewayFQDNProxy) Valid(getter gridtypes.WorkloadGetter) error {
	if !gwNameRegex.MatchString(g.FQDN) {
		return fmt.Errorf("invalid name")
	}
	if g.FQDN[len(g.FQDN)-1] == '.' {
		return fmt.Errorf("fqdn can't end with a dot")
	}
	if len(g.Backends) == 0 {
		return fmt.Errorf("backends list can not be empty")
	}
	for _, backend := range g.Backends {
		if err := backend.Valid(); err != nil {
			return errors.Wrapf(err, "failed to validate backend '%s'", backend)
		}
	}

	return nil
}

func (g GatewayFQDNProxy) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", g.FQDN); err != nil {
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

func (g GatewayFQDNProxy) Capacity() (gridtypes.Capacity, error) {
	// this has to be calculated per bytes served over the gw. so
	// a special handler in reporting that need to calculate and report
	// this.
	return gridtypes.Capacity{}, nil
}

// GatewayProxyResult results
type GatewayProxyResult struct {
	FQDN string `json:"fqdn"`
}
