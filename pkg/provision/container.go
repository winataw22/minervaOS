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
	NetworkID pkg.NetID `json:"network_id"`
	// IP to give to the container
	IPs       []net.IP `json:"ips"`
	PublicIP6 bool     `json:"public_ip6"`
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
	// Env env variables to container that the value is encrypted
	// with the node public key. the env will be exposed to plain
	// text to the entrypoint.
	SecretEnv map[string]string `json:"secret_env"`
	// Entrypoint the process to start inside the container
	Entrypoint string `json:"entrypoint"`
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool `json:"interactive"`
	// Mounts extra mounts in the container
	Mounts []Mount `json:"mounts"`
	// Network network info for container
	Network Network `json:"network"`
	// ContainerCapacity is the amount of resource to allocate to the container
	Capacity ContainerCapacity `json:"capacity"`
}

// ContainerResult is the information return to the BCDB
// after deploying a container
type ContainerResult struct {
	ID   string `json:"id"`
	IPv6 string `json:"ipv6"`
	IPv4 string `json:"ipv4"`
}

// ContainerCapacity is the amount of resource to allocate to the container
type ContainerCapacity struct {
	// Number of CPU
	CPU uint `json:"cpu"`
	// Memory in MiB
	Memory uint64 `json:"memory"`
}

func containerProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	return containerProvisionImpl(ctx, reservation)
}

// ContainerProvision is entry point to container reservation
func containerProvisionImpl(ctx context.Context, reservation *Reservation) (ContainerResult, error) {
	client := GetZBus(ctx)
	cache := GetOwnerCache(ctx)

	containerClient := stubs.NewContainerModuleStub(client)
	flistClient := stubs.NewFlisterStub(client)
	storageClient := stubs.NewStorageModuleStub(client)

	tenantNS := fmt.Sprintf("ns%s", reservation.User)
	containerID := reservation.ID

	var config Container
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return ContainerResult{}, err
	}

	// check if workload is already deployed
	_, err := containerClient.Inspect(tenantNS, pkg.ContainerID(containerID))
	if err == nil {
		log.Info().Str("id", containerID).Msg("container already deployed")
		return ContainerResult{
			ID:   containerID,
			IPv4: config.Network.IPs[0].String(),
		}, nil
	}

	if err := validateContainerConfig(config); err != nil {
		return ContainerResult{}, errors.Wrap(err, "container provision schema not valid")
	}

	log.Debug().Str("flist", config.FList).Msg("mounting flist")
	mnt, err := flistClient.Mount(config.FList, config.FlistStorage, pkg.DefaultMountOptions)
	if err != nil {
		return ContainerResult{}, err
	}

	var env []string
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range config.SecretEnv {
		v, err := decryptSecret(client, v)
		if err != nil {
			return ContainerResult{}, errors.Wrapf(err, "failed to decrypt secret env var '%s'", k)
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	var mounts []pkg.MountInfo
	for _, mount := range config.Mounts {

		owner, err := cache.OwnerOf(mount.VolumeID)
		if err != nil {
			return ContainerResult{}, errors.Wrapf(err, "failed to retrieve the owner of volume %s", mount.VolumeID)
		}

		if owner != reservation.User {
			return ContainerResult{}, fmt.Errorf("cannot use volume %s, user %s is not the owner of it", mount.VolumeID, reservation.User)
		}

		// we make sure that mountpoint in config doesn't have relative parts
		mountpoint := path.Join("/", mount.Mountpoint)

		if err := os.MkdirAll(path.Join(mnt, mountpoint), 0755); err != nil {
			return ContainerResult{}, err
		}

		source, err := storageClient.Path(mount.VolumeID)
		if err != nil {
			return ContainerResult{}, errors.Wrapf(err, "failed to get the mountpoint path of the volume %s", mount.VolumeID)
		}

		mounts = append(
			mounts,
			pkg.MountInfo{
				Source: source,
				Target: mountpoint,
			},
		)
	}

	netID := networkID(reservation.User, string(config.Network.NetworkID))
	log.Debug().
		Str("network-id", string(netID)).
		Str("config", fmt.Sprintf("%+v", config)).
		Msg("deploying network")

	networkMgr := stubs.NewNetworkerStub(GetZBus(ctx))

	ips := make([]string, len(config.Network.IPs))
	for i, ip := range config.Network.IPs {
		ips[i] = ip.String()
	}

	join, err := networkMgr.Join(netID, containerID, ips, config.Network.PublicIP6)
	if err != nil {
		return ContainerResult{}, err
	}

	defer func() {
		if err != nil {
			if err := networkMgr.Leave(netID, containerID); err != nil {
				log.Error().Err(err).Msgf("failed leave containrt network namespace")
			}

			if err := flistClient.Umount(mnt); err != nil {
				log.Error().Err(err).Str("path", mnt).Msgf("failed to unmount")
			}
		}
	}()

	log.Info().
		Str("ipv6", join.IPv6.String()).
		Str("ipv4", join.IPv4.String()).
		Str("container", reservation.ID).
		Msg("assigned an IP")

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
			CPU:         config.Capacity.CPU,
			Memory:      config.Capacity.Memory * 1024 * 1024,
		},
	)
	if err != nil {
		return ContainerResult{}, errors.Wrap(err, "error starting container")
	}

	if config.Network.PublicIP6 {
		join.IPv6, err = getIfaceIP(ctx, "pub", join.Namespace)
		if err != nil {
			return ContainerResult{}, errors.Wrap(err, "error reading container ipv6")
		}
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
	if err == nil {
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

	} else {
		log.Error().Err(err).Str("container", string(containerID)).Msg("failed to inspect container for decomission")
	}

	netID := networkID(reservation.User, string(config.Network.NetworkID))
	if _, err := networkMgr.GetSubnet(netID); err == nil { // simple check to make sure the network still exists on the node
		if err := networkMgr.Leave(netID, string(containerID)); err != nil {
			return errors.Wrap(err, "failed to delete container network namespace")
		}
	}

	return nil
}

func validateContainerConfig(config Container) error {
	if config.Network.NetworkID == "" {
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
