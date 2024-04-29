package primitives

import (
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/primitives/gateway"
	"github.com/threefoldtech/zos/pkg/primitives/network"
	"github.com/threefoldtech/zos/pkg/primitives/pubip"
	"github.com/threefoldtech/zos/pkg/primitives/qsfs"
	"github.com/threefoldtech/zos/pkg/primitives/vm"
	"github.com/threefoldtech/zos/pkg/primitives/volume"
	"github.com/threefoldtech/zos/pkg/primitives/zdb"
	"github.com/threefoldtech/zos/pkg/primitives/zlogs"
	"github.com/threefoldtech/zos/pkg/primitives/zmount"
	"github.com/threefoldtech/zos/pkg/provision"
)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) provision.Provisioner {
	managers := map[gridtypes.WorkloadType]provision.Manager{
		zos.ZMountType:           zmount.NewManager(zbus),
		zos.ZLogsType:            zlogs.NewManager(zbus),
		zos.QuantumSafeFSType:    qsfs.NewManager(zbus),
		zos.ZDBType:              zdb.NewManager(zbus),
		zos.NetworkType:          network.NewManager(zbus),
		zos.PublicIPType:         pubip.NewManager(zbus),
		zos.PublicIPv4Type:       pubip.NewManager(zbus), // backward compatibility
		zos.ZMachineType:         vm.NewManager(zbus),
		zos.VolumeType:           volume.NewManager(zbus),
		zos.GatewayNameProxyType: gateway.NewNameManager(zbus),
		zos.GatewayFQDNProxyType: gateway.NewFQDNManager(zbus),
	}

	return provision.NewMapProvisioner(managers)
}
