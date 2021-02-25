package primitives

import (
	"context"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Primitives hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Primitives struct {
	provision.Provisioner
	zbus zbus.Client
}

var _ provision.Provisioner = (*Primitives)(nil)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) *Primitives {
	p := &Primitives{
		zbus: zbus,
	}

	provisioners := map[gridtypes.WorkloadType]provision.DeployFunction{
		zos.ContainerType:  p.containerProvision,
		zos.VolumeType:     p.volumeProvision,
		zos.NetworkType:    p.networkProvision,
		zos.ZDBType:        p.zdbProvision,
		zos.KubernetesType: p.kubernetesProvision,
		zos.PublicIPType:   p.publicIPProvision,
	}
	decommissioners := map[gridtypes.WorkloadType]provision.RemoveFunction{
		zos.ContainerType:  p.containerDecommission,
		zos.VolumeType:     p.volumeDecommission,
		zos.NetworkType:    p.networkDecommission,
		zos.ZDBType:        p.zdbDecommission,
		zos.KubernetesType: p.kubernetesDecomission,
		zos.PublicIPType:   p.publicIPDecomission,
	}

	p.Provisioner = provision.NewMapProvisioner(provisioners, decommissioners)

	return p
}

// RuntimeUpgrade runs upgrade needed when provision daemon starts
func (p *Primitives) RuntimeUpgrade(ctx context.Context) {
	p.upgradeRunningZdb(ctx)
}
