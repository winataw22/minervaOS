package zos

import (
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	gwNameRegex = regexp.MustCompile(`^[a-zA-Z0-9-.]+$`)
)

type Backend string

// Parse accepts http://ip:port, http://ip or ip:port
// checks if backend string is a valid string based on the tlsPassthrough parameter
// ip:port is only valid in case of tlsPassthrough is true
// http://ip:port or http://ip is valid in case of tlsPassthrough is false
func (b Backend) Valid(tlsPassthrough bool) error {
	var hostName string
	if tlsPassthrough {
		host, port, err := net.SplitHostPort(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse backend %s with error: %w", b, err)
		}

		parsedPort, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid port in backend: %s", port)
		}

		if parsedPort > math.MaxUint16 {
			return fmt.Errorf("port '%s' must be <= 65535", port)
		}

		hostName = host
	} else {
		u, err := url.Parse(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse backend with error: %w", err)
		}

		if u.Scheme != "http" {
			return fmt.Errorf("scheme expected to be http")
		}
		hostName = u.Hostname()
	}

	ip := net.ParseIP(hostName)
	if len(ip) == 0 || ip.IsLoopback() {
		return fmt.Errorf("invalid ip address in backend: %s", hostName)
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
		if err := backend.Valid(g.TLSPassthrough); err != nil {
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
