package zos

import (
	"io"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// PublicIP structure
type PublicIP struct{}

// Valid validate public ip input
func (p PublicIP) Valid(getter gridtypes.WorkloadGetter) error {
	return nil
}

// Challenge implementation
func (p PublicIP) Challenge(b io.Writer) error {
	return nil
}

// Capacity implementation
func (p PublicIP) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{IPV4U: 1}, nil
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP gridtypes.IPNet `json:"ip"`

	// Gateway: this fields is only here because we have no idea what is the
	// gateway of that ip without consulting the farmer. Currently this
	// component does not exist. hence as a temporaray solution the user must
	// also provide
	Gateway net.IP `json:"gateway"`
}
