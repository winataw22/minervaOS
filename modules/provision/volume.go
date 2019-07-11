package provision

import (
	"context"
	"encoding/json"

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

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type DiskType `json:"type"`
}

// VolumeProvision is entry point to provision a volume
func VolumeProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	client := GetZBus(ctx)
	var config Volume
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, err
	}

	storageClient := stubs.NewStorageModuleStub(client)

	return storageClient.CreateFilesystem(config.Size, modules.DeviceType(config.Type))
}
