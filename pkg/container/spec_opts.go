package container

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
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

func (c *containerModule) ensureNamespace(ctx context.Context, client *containerd.Client, namespace string) error {
	service := client.NamespaceService()
	namespaces, err := service.List(ctx)
	if err != nil {
		return err
	}

	for _, ns := range namespaces {
		if ns == namespace {
			return nil
		}
	}

	return service.Create(ctx, namespace, nil)
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

func setResolvConf(root string) error {
	const tmp = "nameserver 1.1.1.1\nnameserver 1.0.0.1\n2606:4700:4700::1111\nnameserver 2606:4700:4700::1001\n"

	path := filepath.Join(root, "etc/resolv.conf")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 644)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	defer f.Close()

	if os.IsNotExist(err) {
		_, err = f.WriteString(tmp)
		return err
	}

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		_, err = f.WriteString(tmp)
	}
	return err
}
