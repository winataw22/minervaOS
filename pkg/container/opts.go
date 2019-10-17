package container

import (
	"context"

	"path"

	"github.com/opencontainers/runtime-spec/specs-go"

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
