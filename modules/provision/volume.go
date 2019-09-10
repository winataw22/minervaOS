package provision

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"

	"github.com/threefoldtech/zosv2/modules/stubs"
)

// DiskType defines disk type
type DiskType string

const (
	// HDDDiskType for hdd disks
	HDDDiskType DiskType = "HDD"
	// SSDDiskType for ssd disks
	SSDDiskType DiskType = "SSD"
)

const (
	// gigabyte to byte conversion
	gigabyte = 1024 * 1024 * 1024
)

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type DiskType `json:"type"`
}

// VolumeResult is the information return to the BCDB
// after deploying a volume
type VolumeResult struct {
	ID string `json:"volume_id"`
}

// VolumeProvision is entry point to provision a volume
func volumeProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	client := GetZBus(ctx)
	var config Volume
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, err
	}

	storageClient := stubs.NewStorageModuleStub(client)

	path, err := storageClient.Path(reservation.ID)
	if err == nil {
		log.Info().Str("id", reservation.ID).Msg("volume already deployed")
		return path, nil
	}

	_, err = storageClient.CreateFilesystem(reservation.ID, config.Size*gigabyte, modules.DeviceType(config.Type))

	return VolumeResult{
		ID: reservation.ID,
	}, err
}

func volumeDecommission(ctx context.Context, reservation *Reservation) error {
	client := GetZBus(ctx)
	storageClient := stubs.NewStorageModuleStub(client)

	return storageClient.ReleaseFilesystem(reservation.ID)
}
