package zmount

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*Manager)(nil)
	_ provision.Updater = (*Manager)(nil)
)

// ZMount defines a mount point
type ZMount = zos.ZMount

// ZMountResult types
type ZMountResult = zos.ZMountResult

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

// VolumeProvision is entry point to provision a volume
func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.volumeProvisionImpl(ctx, wl)
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	vdisk := stubs.NewStorageModuleStub(p.zbus)
	return vdisk.DiskDelete(ctx, wl.ID.String())
}

func (p *Manager) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.zMountUpdateImpl(ctx, wl)
}

func (p *Manager) volumeProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (vol ZMountResult, err error) {
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
func (p *Manager) zMountUpdateImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (vol ZMountResult, err error) {
	log.Debug().Msg("updating zmount")
	current, err := provision.GetWorkload(ctx, wl.Name)
	if err != nil {
		// this should not happen but we need to have the check anyway
		return vol, errors.Wrapf(err, "no zmount workload with name '%s' is deployed", wl.Name.String())
	}

	var old ZMount
	if err := json.Unmarshal(current.Data, &old); err != nil {
		return vol, errors.Wrap(err, "failed to decode reservation schema")
	}

	var new ZMount
	if err := json.Unmarshal(wl.Data, &new); err != nil {
		return vol, errors.Wrap(err, "failed to decode reservation schema")
	}

	if new.Size == old.Size {
		return vol, provision.ErrNoActionNeeded
	} else if new.Size < old.Size {
		return vol, provision.UnChanged(fmt.Errorf("not safe to shrink a disk"))
	}

	// now validate that disk is not being used right now
	deployment, err := provision.GetDeployment(ctx)
	if err != nil {
		return vol, provision.UnChanged(errors.Wrap(err, "failed to get deployment"))
	}

	vms := deployment.ByType(zos.ZMachineType)
	log.Debug().Int("count", len(vms)).Msg("found zmachines in deployment")
	for _, vm := range vms {
		// vm not running, no need to check
		if !vm.Result.State.IsOkay() {
			continue
		}

		var data zos.ZMachine
		if err := json.Unmarshal(vm.Data, &data); err != nil {
			return vol, provision.UnChanged(errors.Wrap(err, "failed to load vm information"))
		}

		for _, mnt := range data.Mounts {
			if mnt.Name == wl.Name {
				return vol, provision.UnChanged(fmt.Errorf("disk is mounted, please delete the VM first"))
			}
		}
	}

	log.Debug().Str("disk", wl.ID.String()).Msg("disk is not used, proceed with update")
	vdisk := stubs.NewStorageModuleStub(p.zbus)

	// okay, so no vm is using this disk. time to try resize.
	vol.ID = wl.ID.String()
	_, err = vdisk.DiskResize(ctx, wl.ID.String(), new.Size)
	// we know it's safe to resize the disk, it won't break it so we
	// can be sure we can wrap the error into an unchanged error
	return vol, provision.UnChanged(err)
}
