package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Network struct
type Network struct {
	NetwokID pkg.NetID `json:"network_id"`
	// IP to give to the container
	IPs []net.IP `json:"ips"`
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
	FlistStorage string `json:"flist_storage"`
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

// ContainerResult is the information return to the BCDB
// after deploying a container
type ContainerResult struct {
	ID   string `json:"id"`
	IPv6 string `json:"ipv6"`
	IPv4 string `json:"ipv4"`
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
	_, err := containerClient.Inspect(tenantNS, pkg.ContainerID(containerID))
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

	netID := networkID(reservation.User, string(config.Network.NetwokID))
	log.Debug().
		Str("network-id", string(netID)).
		Str("config", fmt.Sprintf("%+v", config)).
		Msg("deploying network")

	networkMgr := stubs.NewNetworkerStub(GetZBus(ctx))

	ips := make([]string, len(config.Network.IPs))
	for i, ip := range config.Network.IPs {
		ips[i] = ip.String()
	}

	join, err := networkMgr.Join(netID, containerID, ips)
	if err != nil {
		return nil, err
	}

	// TODO: Push IP back to bcdb
	log.Info().
		Str("ipv6", join.IPv6.String()).
		Str("ipv4", join.IPv4.String()).
		Str("container", reservation.ID).
		Msg("assigned an IP")

	log.Debug().Str("flist", config.FList).Msg("mounting flist")
	mnt, err := flistClient.Mount(config.FList, config.FlistStorage)
	if err != nil {
		return nil, err
	}

	var env []string
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	var mounts []pkg.MountInfo
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
			pkg.MountInfo{
				Source:  source,
				Target:  mountpoint,
				Type:    "none",
				Options: []string{"bind"},
			},
		)
	}
	id, err := containerClient.Run(
		tenantNS,
		pkg.Container{
			Name:   containerID,
			RootFS: mnt,
			Env:    env,
			Network: pkg.NetworkInfo{
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
	return ContainerResult{
		ID:   string(id),
		IPv6: join.IPv6.String(),
		IPv4: join.IPv4.String(),
	}, nil
}

func containerDecommission(ctx context.Context, reservation *Reservation) error {
	client := GetZBus(ctx)

	container := stubs.NewContainerModuleStub(client)
	flist := stubs.NewFlisterStub(client)
	networkMgr := stubs.NewNetworkerStub(client)

	tenantNS := fmt.Sprintf("ns%s", reservation.User)
	containerID := pkg.ContainerID(reservation.ID)

	var config Container
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return err
	}

	info, err := container.Inspect(tenantNS, containerID)
	if err != nil {
		return errors.Wrapf(err, "failed to inspect container %s", containerID)
	}

	if err := container.Delete(tenantNS, containerID); err != nil {
		return errors.Wrapf(err, "failed to delete container %s", containerID)
	}

	rootFS := info.RootFS
	if info.Interactive {
		rootFS, err = findRootFS(info.Mounts)
		if err != nil {
			return err
		}
	}

	if err := flist.Umount(rootFS); err != nil {
		return errors.Wrapf(err, "failed to unmount flist at %s", rootFS)
	}

	netID := networkID(reservation.User, string(config.Network.NetwokID))
	if err := networkMgr.Leave(netID, string(containerID)); err != nil {
		return errors.Wrap(err, "failed to delete container network namespace")
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

func findRootFS(mounts []pkg.MountInfo) (string, error) {
	for _, m := range mounts {
		if m.Target == "/sandbox" {
			return m.Source, nil
		}
	}

	return "", fmt.Errorf("rootfs flist mountpoint not found")
}
