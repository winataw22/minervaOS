package primitives

import (
	"context"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Provisioner hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Provisioner struct {
	cache provision.ReservationCache
	zbus  zbus.Client

	Provisioners    map[provision.ReservationType]provision.ProvisionerFunc
	Decommissioners map[provision.ReservationType]provision.DecomissionerFunc
}

// NewProvisioner creates a new 0-OS provisioner
func NewProvisioner(cache provision.ReservationCache, zbus zbus.Client) *Provisioner {
	p := &Provisioner{
		cache: cache,
		zbus:  zbus,
	}
	p.Provisioners = map[provision.ReservationType]provision.ProvisionerFunc{
		ContainerReservation:       p.containerProvision,
		VolumeReservation:          p.volumeProvision,
		NetworkReservation:         p.networkProvision,
		NetworkResourceReservation: p.networkProvision,
		ZDBReservation:             p.zdbProvision,
		DebugReservation:           p.debugProvision,
		KubernetesReservation:      p.kubernetesProvision,
		PublicIPReservation:        p.publicIPProvision,
	}
	p.Decommissioners = map[provision.ReservationType]provision.DecomissionerFunc{
		ContainerReservation:       p.containerDecommission,
		VolumeReservation:          p.volumeDecommission,
		NetworkReservation:         p.networkDecommission,
		NetworkResourceReservation: p.networkDecommission,
		ZDBReservation:             p.zdbDecommission,
		DebugReservation:           p.debugDecommission,
		KubernetesReservation:      p.kubernetesDecomission,
		PublicIPReservation:        p.publicIPDecomission,
	}

	return p
}

// RuntimeUpgrade runs upgrade needed when provision daemon starts
func (p *Provisioner) RuntimeUpgrade(ctx context.Context) {
	p.upgradeRunningZdb(ctx)
}
