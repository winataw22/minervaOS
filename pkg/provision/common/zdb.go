package common

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// DeleteZdbContainer removes a 0-DB container and cleanup all related resources (container, flist, network)
func DeleteZdbContainer(containerID pkg.ContainerID, zbus zbus.Client) error {
	container := stubs.NewContainerModuleStub(zbus)
	flist := stubs.NewFlisterStub(zbus)

	info, err := container.Inspect("zdb", containerID)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to inspect container '%s'", containerID)
	}

	if err := container.Delete("zdb", containerID); err != nil {
		return errors.Wrapf(err, "failed to delete container %s", containerID)
	}

	network := stubs.NewNetworkerStub(zbus)
	if err := network.ZDBDestroy(info.Network.Namespace); err != nil {
		return errors.Wrapf(err, "failed to destroy zdb network namespace")
	}

	if err := flist.Umount(info.RootFS); err != nil {
		return errors.Wrapf(err, "failed to unmount flist at %s", info.RootFS)
	}

	return nil
}
