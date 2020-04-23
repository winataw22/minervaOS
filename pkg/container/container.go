package container

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/runtime/restart"
	"github.com/google/shlex"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"

	"github.com/containerd/containerd/cio"
)

const (
	containerdSock = "/run/containerd/containerd.sock"
)

const (
	defaultMemory = 256 * 1024 * 1204 // 256MiB
	defaultCPU    = 1
)

var (
	ignoreMntTypes = map[string]struct{}{
		"proc":   {},
		"tmpfs":  {},
		"devpts": {},
		"mqueue": {},
		"sysfs":  {},
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

// New return an new pkg.ContainerModule
func New(root string, containerd string) pkg.ContainerModule {
	if len(containerd) == 0 {
		containerd = containerdSock
	}

	return &containerModule{
		containerd: containerd,
		root:       root,
	}
}

// Run creates and starts a container
func (c *containerModule) Run(ns string, data pkg.Container) (id pkg.ContainerID, err error) {
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

	if data.Memory == 0 {
		data.Memory = defaultMemory
	}
	if data.CPU == 0 {
		data.CPU = defaultCPU
	}

	if data.Logs == nil {
		data.Logs = []logger.Logs{}
	}

	// we never allow any container to boot without a network namespace
	if data.Network.Namespace == "" {
		return "", fmt.Errorf("cannot create container without network namespace")
	}

	if err := applyStartup(&data, filepath.Join(data.RootFS, ".startup.toml")); err != nil {
		errors.Wrap(err, "error updating environment variable from startup file")
	}

	opts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/amd64"),
		oci.WithRootFSPath(data.RootFS),
		oci.WithEnv(data.Env),
		oci.WithHostResolvconf,
		removeRunMount(),
		withNetworkNamespace(data.Network.Namespace),
		withMounts(data.Mounts),
		WithMemoryLimit(data.Memory),
		WithCPUCount(data.CPU),
	}

	if data.WorkingDir != "" {
		opts = append(opts, oci.WithProcessCwd(data.WorkingDir))
	}

	if data.Interactive {
		opts = append(opts, withCoreX())
	} else {
		args, err := shlex.Split(data.Entrypoint)
		if err != nil || len(args) == 0 {
			return id, fmt.Errorf("invalid entrypoint definition '%s'", data.Entrypoint)
		}

		opts = append(opts, oci.WithProcessArgs(args...))
	}

	log.Info().
		Str("namespace", ns).
		Str("data", fmt.Sprintf("%+v", data)).
		Msgf("create new container")

	container, err := client.NewContainer(
		ctx,
		data.Name,
		containerd.WithNewSpec(opts...),
		// this ensure that the container/task will be restarted automatically
		// if it gets killed for whatever reason (mostly OOM killer)
		restart.WithStatus(containerd.Running),
	)
	if err != nil {
		return id, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return id, err
	}
	log.Info().Msgf("args %+v", spec.Process.Args)
	log.Info().Msgf("env %+v", spec.Process.Env)
	log.Info().Msgf("root %+v", spec.Root)
	for _, linxNS := range spec.Linux.Namespaces {
		log.Info().Msgf("namespace %+v", linxNS.Type)
	}
	log.Info().Msgf("mounts %+v", spec.Mounts)

	defer func() {
		// if any of the next steps below fails, make sure
		// we delete the container.
		// (preparing, creating, and starting a task)
		if err != nil {
			container.Delete(ctx, containerd.WithSnapshotCleanup)
		}
	}()

	// creating logs config directories
	cfgs := path.Join(c.root, "config", ns)
	if err = os.MkdirAll(cfgs, 0755); err != nil {
		return id, err
	}

	// creating and serializing logs settings for external logger
	confpath := path.Join(cfgs, fmt.Sprintf("%s-logs.json", container.ID()))
	log.Info().Str("cfg", confpath).Msg("writing logs settings")

	err = logger.Serialize(confpath, data.Logs)
	if err != nil {
		log.Error().Err(err).Msg("could not write logs settings")
		return id, err
	}

	// setting external logger process
	uri, err := url.Parse("binary:///bin/shim-logs")
	if err != nil {
		log.Error().Err(err).Msg("log uri")
		return id, err
	}

	log.Info().Str("loguri", uri.String()).Msg("external logging process")

	task, err := container.NewTask(ctx, cio.LogURI(uri))
	if err != nil {
		log.Error().Err(err).Msg("logger new task")
		return id, err
	}

	// set user defined endpoint stats
	for _, l := range data.StatsAggregator {
		switch l.Type {
		case stats.RedisType:
			s, err := stats.NewRedis(l.Data.Endpoint)

			if err != nil {
				log.Error().Err(err).Msg("redis stats")
				continue
			}

			go stats.Monitor(c.containerd, ns, data.Name, s)

		default:
			log.Error().Str("type", l.Type).Msg("invalid stats type requested")
		}
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return id, err
	}

	return pkg.ContainerID(container.ID()), nil
}

// Inspect returns the detail about a running container
func (c *containerModule) Inspect(ns string, id pkg.ContainerID) (result pkg.Container, err error) {
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

	if spec.Root.Path == "/usr/lib/corex" {
		result.Interactive = true
	}

	if process := spec.Process; process != nil {
		result.Entrypoint = strings.Join(process.Args, " ")
		result.Env = process.Env
	}

	for _, mount := range spec.Mounts {
		if _, ok := ignoreMntTypes[mount.Type]; ok {
			continue
		}
		result.Mounts = append(result.Mounts,
			pkg.MountInfo{
				Source: mount.Source,
				Target: mount.Destination,
			},
		)
	}

	for _, namespace := range spec.Linux.Namespaces {
		if namespace.Type == specs.NetworkNamespace {
			result.Network.Namespace = filepath.Base(namespace.Path)
		}
	}

	return
}

// Deletes stops and remove a container
func (c *containerModule) Delete(ns string, id pkg.ContainerID) error {
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

	if err := container.Update(ctx, restart.WithNoRestarts); err != nil {
		log.Warn().Err(err).Msg("failed to clear up restart task status, continuing anyways")
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

func applyStartup(data *pkg.Container, path string) error {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		log.Info().Msg("startup file found")

		startup := startup{}
		if _, err := toml.DecodeReader(f, &startup); err != nil {
			return err
		}

		entry, ok := startup.Entries["entry"]
		if !ok {
			return nil
		}

		data.Env = mergeEnvs(entry.Envs(), data.Env)
		if data.Entrypoint == "" && entry.Entrypoint() != "" {
			data.Entrypoint = entry.Entrypoint()
		}
		if data.WorkingDir == "" && entry.WorkingDir() != "" {
			data.WorkingDir = entry.WorkingDir()
		}
	}
	return nil
}
