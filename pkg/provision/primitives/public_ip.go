package primitives

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// PublicIP structure
type PublicIP struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IPNet `json:"ip"`
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

func (p *Provisioner) publicIPProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.publicIPProvisionImpl(ctx, reservation)
}

func (p *Provisioner) publicIPProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result PublicIPResult, err error) {
	return PublicIPResult{}, nil
}

func (p *Provisioner) publicIPDecomission(ctx context.Context, reservation *provision.Reservation) error {
	// Disconnect the public interface from the network if one exists
	network := stubs.NewNetworkerStub(p.zbus)
	return network.DisconnectPubTap(reservation.ID)
}
