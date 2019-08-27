package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

// Network struct
type Network struct {
	NetwokID string `json:"network_id"`
}

// Mount defines a container volume mounted inside the container
type Mount struct {
	VolumeID   string `json:"volume_id"`
	Mountpoint string `json:"mountpoint"`
}

//Container creation info
type Container struct {
	// URL of the flist
	FList string `json:"flist"`
	// URL of the storage backend for the flist
	FlistStorage string `json:"flist"`
	// Env env variables to container in format
	Env map[string]string `json:"env"`
	// Entrypoint the process to start inside the container
	Entrypoint string `json:"entrypoint"`
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool `json:"interactive"`
	// Mounts extra mounts in the container
	Mounts []Mount `json:"mounts"`
	// Network network info for container
	Network Network `json:"network"`
}

// ContainerProvision is entry point to container reservation
func containerProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	client := GetZBus(ctx)
	cache := GetOwnerCache(ctx)

	containerClient := stubs.NewContainerModuleStub(client)
	flistClient := stubs.NewFlisterStub(client)
	storageClient := stubs.NewStorageModuleStub(client)

	tenantNS := fmt.Sprintf("ns%s", reservation.User)
	containerID := reservation.ID

	// check if workload is already deployed
	_, err := containerClient.Inspect(tenantNS, modules.ContainerID(containerID))
	if err == nil {
		log.Info().Str("id", containerID).Msg("container already deployed")
		return containerID, nil
	}

	var config Container
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, err
	}

	if err := validateContainerConfig(config); err != nil {
		return nil, errors.Wrap(err, "container provision schema not valid")
	}

	log.Debug().
		Str("network-id", config.Network.NetwokID).
		Str("config", fmt.Sprintf("%+v", config)).
		Msg("deploying network")

	networkMgr := stubs.NewNetworkerStub(GetZBus(ctx))
	join, err := networkMgr.Join(reservation.ID, modules.NetID(config.Network.NetwokID))
	if err != nil {
		return nil, err
	}

	// TODO: Push IP back to bcdb
	log.Info().Str("ip", join.IP.String()).Str("container", reservation.ID).Msg("assigned an IP")

	log.Debug().Str("flist", config.FList).Msg("mounting flist")
	mnt, err := flistClient.Mount(config.FList, config.FlistStorage)
	if err != nil {
		return nil, err
	}

	var env []string
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	var mounts []modules.MountInfo
	for _, mount := range config.Mounts {

		owner, err := cache.OwnerOf(mount.VolumeID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve the owner of volume %s", mount.VolumeID)
		}

		if owner != reservation.User {
			return nil, fmt.Errorf("cannot use volume %s, user %s is not the owner of it", mount.VolumeID, reservation.User)
		}

		// we make sure that mountpoint in config doesn't have relative parts
		mountpoint := path.Join("/", mount.Mountpoint)

		if err := os.MkdirAll(path.Join(mnt, mountpoint), 0755); err != nil {
			return nil, err
		}

		source, err := storageClient.Path(mount.VolumeID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get the mountpoint path of the volume %s", mount.VolumeID)
		}

		mounts = append(
			mounts,
			modules.MountInfo{
				Source:  source,
				Target:  mountpoint,
				Type:    "none",
				Options: []string{"bind"},
			},
		)
	}
	id, err := containerClient.Run(
		tenantNS,
		modules.Container{
			Name:   reservation.ID,
			RootFS: mnt,
			Env:    env,
			Network: modules.NetworkInfo{
				Namespace: join.Namespace,
			},
			Mounts:      mounts,
			Entrypoint:  config.Entrypoint,
			Interactive: config.Interactive,
		},
	)

	if err != nil {
		if err := flistClient.Umount(mnt); err != nil {
			log.Error().Err(err).Str("path", mnt).Msgf("failed to unmount")
		}

		return nil, err
	}

	log.Info().Msgf("container created with id: '%s'", id)
	return id, nil
}

func containerDecommission(ctx context.Context, reservation *Reservation) error {
	client := GetZBus(ctx)

	container := stubs.NewContainerModuleStub(client)
	flist := stubs.NewFlisterStub(client)

	id := modules.ContainerID(reservation.ID)

	info, err := container.Inspect(reservation.User, id)
	if err != nil {
		return errors.Wrapf(err, "failed to inspect container %s", id)
	}

	if err := container.Delete(reservation.User, modules.ContainerID(id)); err != nil {
		return errors.Wrapf(err, "failed to delete container %s", id)
	}

	if err := flist.Umount(info.RootFS); err != nil {
		return errors.Wrapf(err, "failed to unmount flist at %s", info.RootFS)
	}
	return nil
}

func validateContainerConfig(config Container) error {
	if config.Network.NetwokID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if config.FList == "" {
		return fmt.Errorf("missing flist url")
	}

	return nil
}
