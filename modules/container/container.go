package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/shlex"
	"github.com/threefoldtech/zosv2/modules"
)

const (
	containerdSock = "/run/containerd/containerd.sock"
)

var (
	ignoreMntTypes = map[string]struct{}{
		"proc":   struct{}{},
		"tmpfs":  struct{}{},
		"devpts": struct{}{},
		"mqueue": struct{}{},
		"sysfs":  struct{}{},
	}
)

var (
	// ErrEmptyRootFS is returned when RootFS field is empty when trying to create a container
	ErrEmptyRootFS = errors.New("RootFS of the container creation data cannot be empty")
)

type containerModule struct {
	containerd string
	root       string
}

// New return an new modules.ContainerModule
func New(root string, containerd string) modules.ContainerModule {
	if len(containerd) == 0 {
		containerd = containerdSock
	}

	return &containerModule{
		containerd: containerd,
		root:       root,
	}
}

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

// Run creates and starts a container
// THIS IS A WIP Create action and it's not fully implemented atm
func (c *containerModule) Run(ns string, data modules.Container) (id modules.ContainerID, err error) {
	log.Info().
		Str("namesapce", ns).
		Str("data", fmt.Sprintf("%+v", data)).
		Msgf("create new container")
	// create a new client connected to the default socket path for containerd
	client, err := containerd.New(c.containerd)
	if err != nil {
		return id, err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	if err := c.ensureNamespace(ctx, client, ns); err != nil {
		return id, err
	}

	if data.RootFS == "" {
		return id, ErrEmptyRootFS
	}

	if data.Interactive {
		if err := os.MkdirAll(filepath.Join(data.RootFS, "sandbox"), 0770); err != nil {
			return id, err
		}
		data.Mounts = append(data.Mounts, modules.MountInfo{
			Source:  data.RootFS,
			Target:  "/sandbox",
			Type:    "bind",
			Options: []string{"rbind"}, // mount options
		})
		data.RootFS = "/usr/lib/corex"
		data.Entrypoint = "/bin/corex --chroot /sandbox -d 7"
	}

	args, err := shlex.Split(data.Entrypoint)
	if err != nil || len(args) == 0 {
		return id, fmt.Errorf("invalid entrypoint definition '%s'", data.Entrypoint)
	}

	opts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/amd64"),
		oci.WithRootFSPath(data.RootFS),
		oci.WithProcessArgs(args...),
		oci.WithEnv(data.Env),
	}
	if data.Interactive {
		opts = append(
			opts,
			withAddedCapabilities([]string{
				"CAP_SYS_ADMIN",
			}),
			// in interactive mode, since we start the container
			// from /usr/lib/corex
			// we make it read-only
			oci.WithReadonlyPaths([]string{"/"}),
		)
	}

	if data.Network.Namespace != "" {
		opts = append(
			opts,
			withNetworkNamespace(data.Network.Namespace),
		)
	}

	for _, mount := range data.Mounts {
		opts = append(
			opts,
			oci.WithMounts([]specs.Mount{
				{
					Destination: mount.Target,
					Type:        mount.Type,
					Source:      mount.Source,
					Options:     mount.Options,
				},
			}),
		)
	}

	container, err := client.NewContainer(
		ctx,
		data.Name,
		containerd.WithNewSpec(opts...),
	)
	if err != nil {
		return id, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return id, err
	}
	log.Info().Msgf("args %+v", spec.Process.Args)
	log.Info().Msgf("root %+v", spec.Root)
	for _, ns := range spec.Linux.Namespaces {
		log.Info().Msgf("namespace %+v", ns.Type)

	}

	defer func() {
		// if any of the next steps below fails, make sure
		// we delete the container.
		// (preparing, creating, and starting a task)
		if err != nil {
			container.Delete(ctx, containerd.WithSnapshotCleanup)
		}
	}()

	logs := path.Join(c.root, "logs", ns)
	if err = os.MkdirAll(logs, 0755); err != nil {
		return id, err
	}

	task, err := container.NewTask(ctx, cio.LogFile(path.Join(logs, fmt.Sprintf("%s.log", container.ID()))))
	if err != nil {
		return id, err
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return id, err
	}

	return modules.ContainerID(container.ID()), nil
}

// Inspect returns the detail about a running container
func (c *containerModule) Inspect(ns string, id modules.ContainerID) (result modules.Container, err error) {
	client, err := containerd.New(c.containerd)
	if err != nil {
		return result, err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	container, err := client.LoadContainer(ctx, string(id))
	if err != nil {
		return result, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return result, err
	}

	result.RootFS = spec.Root.Path
	result.Name = container.ID()
	if process := spec.Process; process != nil {
		result.Entrypoint = strings.Join(process.Args, " ")
		result.Env = process.Env
	}

	for _, mount := range spec.Mounts {
		if _, ok := ignoreMntTypes[mount.Type]; ok {
			continue
		}
		result.Mounts = append(result.Mounts,
			modules.MountInfo{
				Source:  mount.Source,
				Target:  mount.Destination,
				Type:    mount.Type,
				Options: mount.Options,
			},
		)
	}

	for _, namespace := range spec.Linux.Namespaces {
		if namespace.Type == "network" {
			result.Network.Namespace = namespace.Path
		}
	}

	return
}

// Deletes stops and remove a container
func (c *containerModule) Delete(ns string, id modules.ContainerID) error {
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	container, err := client.LoadContainer(ctx, string(id))
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err == nil {
		// err == nil, there is a task running inside the container
		exitC, err := task.Wait(ctx)
		if err != nil {
			return err
		}
		trials := 3
	loop:
		for {
			signal := syscall.SIGTERM
			if trials <= 0 {
				signal = syscall.SIGKILL
			}
			_ = task.Kill(ctx, signal)
			trials--
			select {
			case <-exitC:
				break loop
			case <-time.After(1 * time.Second):
			}
		}

		if _, err := task.Delete(ctx); err != nil {
			return err
		}
	}

	return container.Delete(ctx)
}
