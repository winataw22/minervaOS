package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
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
func ContainerProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	client := GetZBus(ctx)
	containerClient := stubs.NewContainerModuleStub(client)
	flistClient := stubs.NewFlisterStub(client)

	var config Container
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, err
	}

	log.Debug().
		Str("network-id", config.Network.NetwokID).
		Str("config", fmt.Sprintf("%+v", config)).
		Msg("deploying network")

	networkMgr := stubs.NewNetworkerStub(GetZBus(ctx))
	ns, err := networkMgr.Namespace(modules.NetID(config.Network.NetwokID))
	if err != nil {
		return nil, err
	}

	log.Debug().Str("flist", config.FList).Msg("mounting flist")
	mnt, err := flistClient.Mount(config.FList, "")
	if err != nil {
		return nil, err
	}
	var env []string
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	var mounts []modules.MountInfo
	for _, mount := range config.Mounts {
		// we make sure that mountpoint in config doesn't have relative parts
		mountpoint := path.Join("/", mount.Mountpoint)

		if err := os.MkdirAll(path.Join(mnt, mountpoint), 0755); err != nil {
			return nil, err
		}

		mounts = append(
			mounts,
			modules.MountInfo{
				Source:  mount.VolumeID,
				Target:  mountpoint,
				Type:    "none",
				Options: []string{"bind"},
			},
		)
	}

	id, err := containerClient.Run(
		reservation.Tenant.String(),
		modules.Container{
			Name:   uuid.New().String(),
			RootFS: mnt,
			Env:    env,
			Network: modules.NetworkInfo{
				Namespace: ns,
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
