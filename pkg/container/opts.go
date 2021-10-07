package container

import (
	"context"
	"fmt"

	"path"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/shirou/gopsutil/cpu"
	"github.com/threefoldtech/zos/pkg"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
)

// Defines the container cpu period
const period = 100000

// withNetworkNamespace set the named network namespace to use for the container
func withNetworkNamespace(name string) oci.SpecOpts {
	return oci.WithLinuxNamespace(
		specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: path.Join("/var/run/netns", name),
		},
	)
}

func removeRunMount() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		for i, mount := range s.Mounts {
			if mount.Destination == "/run" {
				s.Mounts = append(s.Mounts[:i], s.Mounts[i+1:]...)
				break
			}
		}
		return nil
	}
}

// withCoreX enable corex in a container
// to do so, it mounts the corex binary into the container and set the entrypoint
func withCoreX() oci.SpecOpts {

	withMount := func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: "/corex",
			Type:        "bind",
			Source:      "/usr/bin/corex",
			Options:     []string{"rbind", "ro"},
		})
		return nil
	}

	return oci.Compose(withMount, oci.WithProcessArgs("/corex", "--ipv6", "-d", "7", "--interface", "eth0"))
}

func withMounts(mounts []pkg.MountInfo) oci.SpecOpts {
	mnts := make([]specs.Mount, len(mounts))

	for i, mount := range mounts {
		mnts[i] = specs.Mount{
			Destination: mount.Target,
			Type:        "bind",
			Source:      mount.Source,
			Options:     []string{"rbind"},
		}
	}
	return oci.WithMounts(mnts)
}

// WithRootfsPropagation makes the
func WithRootfsPropagation(rootfsPropagation pkg.RootFSPropagation) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		s.Linux.RootfsPropagation = string(rootfsPropagation)
		return nil
	}
}

// WithMemoryLimit sets the `Linux.LinuxResources.Memory.Limit` section to the
// `limit` specified if the `Linux` section is not `nil`. Additionally sets the
// `Windows.WindowsResources.Memory.Limit` section if the `Windows` section is
// not `nil`.
func WithMemoryLimit(limit uint64) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		if s.Linux != nil {
			if s.Linux.Resources == nil {
				s.Linux.Resources = &specs.LinuxResources{}
			}
			if s.Linux.Resources.Memory == nil {
				s.Linux.Resources.Memory = &specs.LinuxMemory{}
			}
			l := int64(limit)
			s.Linux.Resources.Memory.Limit = &l
		}
		if s.Windows != nil {
			if s.Windows.Resources == nil {
				s.Windows.Resources = &specs.WindowsResources{}
			}
			if s.Windows.Resources.Memory == nil {
				s.Windows.Resources.Memory = &specs.WindowsMemoryResources{}
			}
			s.Windows.Resources.Memory.Limit = &limit
		}
		return nil
	}
}

// WithCPUCount configure the CPU cgroup to limit the amount of CPU used by the container
func WithCPUCount(cru uint) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		totalCPU, err := cpu.Counts(true)
		if err != nil {
			return err
		}

		if cru > uint(totalCPU) {
			return fmt.Errorf("asked %d CRU while only %d are available", cru, totalCPU)
		}

		quota, period := cruToLimit(cru)

		if s.Linux.Resources == nil {
			s.Linux.Resources = &specs.LinuxResources{}
		}
		if s.Linux.Resources.CPU == nil {
			s.Linux.Resources.CPU = &specs.LinuxCPU{
				Quota:  &quota,
				Period: &period,
			}
		}

		return nil
	}
}

func cruToLimit(cru uint) (int64, uint64) {
	quota := int64(period * uint64(cru))

	return quota, period
}
