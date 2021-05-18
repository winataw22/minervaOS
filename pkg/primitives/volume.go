package primitives

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Volume defines a mount point
type Volume = zos.Volume

// VolumeResult types
type VolumeResult = zos.VolumeResult

func (p *Primitives) volumeProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (vol VolumeResult, err error) {
	var config Volume
	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return VolumeResult{}, err
	}

	vol.ID = wl.ID.String()
	storageClient := stubs.NewStorageModuleStub(p.zbus)
	size := config.Size
	fs, err := storageClient.Path(ctx, wl.ID.String())
	if err == nil {
		// todo: validate that the volume type has not been changed.
		// filesystem exist. do we need to (resize)
		log.Info().Stringer("id", wl.ID).Msg("volume already deployed")
		if fs.Usage.Size != size {
			log.Info().Stringer("id", wl.ID).Uint64("size", uint64(size)).Msg("resizing volume")
			_, err := storageClient.UpdateFilesystem(ctx, wl.ID.String(), size)
			if err != nil {
				return vol, err
			}
		}
		return vol, err
	}

	_, err = storageClient.CreateFilesystem(ctx, wl.ID.String(), size, config.Type)

	return vol, err
}

// VolumeProvision is entry point to provision a volume
func (p *Primitives) volumeProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.volumeProvisionImpl(ctx, wl)
}

func (p *Primitives) volumeDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	storageClient := stubs.NewStorageModuleStub(p.zbus)

	return storageClient.ReleaseFilesystem(ctx, wl.ID.String())
}
