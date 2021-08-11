package primitives

import (
	"context"
	"encoding/json"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// ZMount defines a mount point
type ZMount = zos.ZMount

// ZMountResult types
type ZMountResult = zos.ZMountResult

func (p *Primitives) volumeProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (vol ZMountResult, err error) {
	var config ZMount
	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return ZMountResult{}, err
	}

	vol.ID = wl.ID.String()
	vdisk := stubs.NewStorageModuleStub(p.zbus)
	if vdisk.DiskExists(ctx, vol.ID) {
		return vol, nil
	}

	_, err = vdisk.DiskCreate(ctx, vol.ID, config.Size)

	return vol, err
}

// VolumeProvision is entry point to provision a volume
func (p *Primitives) zMountProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.volumeProvisionImpl(ctx, wl)
}

func (p *Primitives) zMountDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	vdisk := stubs.NewStorageModuleStub(p.zbus)
	return vdisk.DiskDelete(ctx, wl.ID.String())
}
