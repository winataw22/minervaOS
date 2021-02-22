package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// PublicIP structure
type PublicIP struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP gridtypes.IPNet `json:"ip"`
}

// Valid validate public ip input
func (p PublicIP) Valid() error {
	if len(p.IP.IP) == 0 {
		return fmt.Errorf("empty ip value")
	}

	if p.IP.IP.To4() == nil {
		return fmt.Errorf("invalid ip format")
	}

	return nil
}

// Challenge implementation
func (p PublicIP) Challenge(b io.Writer) error {
	_, err := fmt.Fprintf(b, "%v", p.IP.String())
	return err
}

// Capacity implementation
func (p PublicIP) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{IPV4U: 1}, nil
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	IP gridtypes.IPNet `json:"ip"`
}
