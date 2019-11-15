package container

import (
	"context"

	"path"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/threefoldtech/zos/pkg"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
)

// withNetworkNamespace set the named network namespace to use for the container
func withNetworkNamespace(name string) oci.SpecOpts {
	return oci.WithLinuxNamespace(
		specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: path.Join("/var/run/netns", name),
		},
	)
}

func withHooks(hooks specs.Hooks) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, spec *oci.Spec) error {
		spec.Hooks = &hooks
		return nil
	}
}

func capsContain(caps []string, s string) bool {
	for _, c := range caps {
		if c == s {
			return true
		}
	}
	return false
}

func withAddedCapabilities(caps []string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		// setCapabilities(s)
		for _, c := range caps {
			for _, cl := range []*[]string{
				&s.Process.Capabilities.Bounding,
				&s.Process.Capabilities.Effective,
				&s.Process.Capabilities.Permitted,
				&s.Process.Capabilities.Inheritable,
			} {
				if !capsContain(*cl, c) {
					*cl = append(*cl, c)
				}
			}
		}
		return nil
	}
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

	return oci.Compose(withMount, oci.WithProcessArgs("/corex", "--ipv6", "-d", "7"))
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
	return oci.Compose(oci.WithMounts(mnts))
}
