package primitives

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// gigabyte to byte conversion
	gigabyte uint64 = 1024 * 1024 * 1024
)

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type pkg.DeviceType `json:"type"`
}

// VolumeResult is the information return to the BCDB
// after deploying a volume
type VolumeResult struct {
	ID string `json:"volume_id"`
}

func (p *Provisioner) volumeProvisionImpl(ctx context.Context, reservation *provision.Reservation) (VolumeResult, error) {
	var config Volume
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return VolumeResult{}, err
	}

	storageClient := stubs.NewStorageModuleStub(p.zbus)

	_, err := storageClient.Path(reservation.ID)
	if err == nil {
		log.Info().Str("id", reservation.ID).Msg("volume already deployed")
		return VolumeResult{
			ID: reservation.ID,
		}, nil
	}

	_, err = storageClient.CreateFilesystem(reservation.ID, config.Size*gigabyte, config.Type)

	return VolumeResult{
		ID: reservation.ID,
	}, err
}

// VolumeProvision is entry point to provision a volume
func (p *Provisioner) volumeProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.volumeProvisionImpl(ctx, reservation)
}

func (p *Provisioner) volumeDecommission(ctx context.Context, reservation *provision.Reservation) error {
	storageClient := stubs.NewStorageModuleStub(p.zbus)

	return storageClient.ReleaseFilesystem(reservation.ID)
}
