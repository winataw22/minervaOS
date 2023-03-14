package zos

import (
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type Backend string

// Parse accepts http://ip:port, http://ip or ip:port
// checks if backend string is a valid string based on the tlsPassthrough parameter
// ip:port is only valid in case of tlsPassthrough is true
// http://ip:port or http://ip is valid in case of tlsPassthrough is false
func (b Backend) Valid(tlsPassthrough bool) error {
	if tlsPassthrough {
		_, _, err := asIpPort(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse backend %s with error: %w", b, err)
		}

	} else {
		u, err := url.Parse(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse backend with error: %w", err)
		}

		if u.Scheme != "http" {
			return fmt.Errorf("scheme expected to be http")
		}

		ip := net.ParseIP(u.Hostname())
		if len(ip) == 0 || ip.IsLoopback() {
			return fmt.Errorf("invalid ip address in backend: %s", u.Hostname())
		}
	}

	return nil
}

func asIpPort(a string) (ip net.IP, port uint16, err error) {
	h, p, err := net.SplitHostPort(a)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse backend %s with error: %w", a, err)
	}

	if ip = net.ParseIP(h); ip == nil {
		return nil, 0, fmt.Errorf("invalid ip")
	}

	parsedPort, err := strconv.ParseUint(p, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid port in backend: %s", p)
	}

	if parsedPort > math.MaxUint16 {
		return nil, 0, fmt.Errorf("port '%s' must be <= 65535", p)
	}

	port = uint16(parsedPort)
	return
}

func (b Backend) AsAddress() (string, error) {
	if _, _, err := asIpPort(string(b)); err == nil {
		return string(b), nil
	}

	// otherwise it must be a url
	u, err := url.Parse(string(b))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse backend as a url")
	}

	if u.Port() == "" {
		return fmt.Sprintf("%s:80", u.Host), nil
	}

	return u.Host, nil
}

// GatewayBase definition. this will proxy name.<zos.domain> to backends
type GatewayBase struct {
	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool `json:"tls_passthrough"`

	// Backends are list of backend ips (only one is supported atm)
	Backends []Backend `json:"backends"`

	// Network name to join [optional].
	// If set the backend IP can be a private ip in that network.
	// the network then must be
	// the same rules for tls-passthrough applies.
	Network *gridtypes.Name `json:"network,omitempty"`
}

func (g GatewayBase) Valid(getter gridtypes.WorkloadGetter) error {
	if len(g.Backends) == 0 {
		return fmt.Errorf("backends list can not be empty")
	}

	if len(g.Backends) != 1 {
		return fmt.Errorf("only one backend is supported")
	}

	for _, backend := range g.Backends {
		if err := backend.Valid(g.TLSPassthrough); err != nil {
			return errors.Wrapf(err, "failed to validate backend '%s'", backend)
		}
	}

	return nil
}

func (g GatewayBase) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%t", g.TLSPassthrough); err != nil {
		return err
	}

	for _, backend := range g.Backends {
		if _, err := fmt.Fprintf(w, "%s", string(backend)); err != nil {
			return err
		}
	}

	if g.Network != nil {
		if _, err := fmt.Fprintf(w, "%s", *g.Network); err != nil {
			return err
		}
	}

	return nil
}
