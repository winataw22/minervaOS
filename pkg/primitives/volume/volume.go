package volume

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type Volume = zos.Volume
type VolumeResult = zos.VolumeResult

type Manager struct {
	client zbus.Client
}

var (
	_ provision.Manager = (*Manager)(nil)
	_ provision.Updater = (*Manager)(nil)
)

func NewManager(client zbus.Client) Manager {
	return Manager{client: client}
}

func (m Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	storage := stubs.NewStorageModuleStub(m.client)

	var volume Volume
	if err := json.Unmarshal(wl.Data, &volume); err != nil {
		return nil, fmt.Errorf("failed to parse workload data as volume: %w", err)
	}
	volumeName := wl.ID.String()

	exists, err := storage.VolumeExists(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup volume with name %q: %w", volumeName, err)
	} else if exists {
		return VolumeResult{ID: volumeName}, provision.ErrNoActionNeeded
	}

	vol, err := storage.VolumeCreate(ctx, volumeName, volume.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to create new volume with name %q: %w", volumeName, err)
	}
	return VolumeResult{ID: vol.Name}, nil
}
func (m Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	storage := stubs.NewStorageModuleStub(m.client)

	var volume Volume
	if err := json.Unmarshal(wl.Data, &volume); err != nil {
		return fmt.Errorf("failed to parse workload data as volume: %w", err)
	}

	volumeName := wl.ID.String()

	exists, err := storage.VolumeExists(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("failed to lookup volume %q: %w", volumeName, err)
	} else if !exists {
		return fmt.Errorf("no volume with name %q found: %w", volumeName, err)
	}

	return storage.VolumeDelete(ctx, volumeName)
}

func (m Manager) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	storage := stubs.NewStorageModuleStub(m.client)

	var volume Volume
	if err := json.Unmarshal(wl.Data, &volume); err != nil {
		return nil, fmt.Errorf("failed to parse workload data as volume: %w", err)
	}
	volumeName := wl.ID.String()

	exists, err := storage.VolumeExists(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup volume %q: %w", volumeName, err)
	} else if !exists {
		return nil, fmt.Errorf("no volume with name %q found: %w", volumeName, err)
	}

	vol, err := storage.VolumeLookup(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup volume %q: %w", volumeName, err)
	}

	if volume.Size < vol.Usage.Size {
		return nil, fmt.Errorf("cannot shrink volume to be less than provisioned space. old: %d, requested: %d", vol.Usage.Size, volume.Size)
	}

	if err := storage.VolumeUpdate(ctx, volumeName, volume.Size); err != nil {
		return nil, fmt.Errorf("failed to update volume %q: %w", volumeName, err)
	}
	return VolumeResult{ID: volumeName}, nil
}
