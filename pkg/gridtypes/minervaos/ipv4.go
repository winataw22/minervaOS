package zos

import (
	"fmt"
	"io"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// PublicIP4 structure
// this is a deprecated type and only kept here for backward compatibility with
// older deployments.
// Please use PublicIP instead
type PublicIP4 struct{}

// Valid validate public ip input
func (p PublicIP4) Valid(getter gridtypes.WorkloadGetter) error {
	return nil
}

// Challenge implementation
func (p PublicIP4) Challenge(b io.Writer) error {
	return nil
}

// Capacity implementation
func (p PublicIP4) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{IPV4U: 1}, nil
}

type PublicIP struct {
	// V4 use one of the reserved Ipv4 from your contract. The Ipv4
	// itself costs money + the network traffic
	V4 bool `json:"v4"`
	// V6 get an ipv6 for the VM. this is for free
	// but the consumed capacity (network traffic) is not
	V6 bool `json:"v6"`
}

// Valid validate public ip input
func (p PublicIP) Valid(getter gridtypes.WorkloadGetter) error {
	if !p.V4 && !p.V6 {
		return fmt.Errorf("public ip workload with no selections")
	}

	return nil
}

// Challenge implementation
func (p PublicIP) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%t", p.V4); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", p.V6); err != nil {
		return err
	}

	return nil
}

// Capacity implementation
func (p PublicIP) Capacity() (gridtypes.Capacity, error) {
	var c uint64
	if p.V4 {
		c = 1
	}
	return gridtypes.Capacity{IPV4U: c}, nil
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP gridtypes.IPNet `json:"ip"`
	// IPv6 of the VM.
	IPv6 gridtypes.IPNet `json:"ip6"`
	// Gateway: this fields is only here because we have no idea what is the
	// gateway of that ip without consulting the farmer. Currently this
	// component does not exist. hence as a temporaray solution the user must
	// also provide
	Gateway net.IP `json:"gateway"`
}
