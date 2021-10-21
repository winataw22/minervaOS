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
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/runtime/restart"
	"github.com/google/shlex"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"

	"github.com/containerd/containerd/cio"
)

const (
	containerdSock = "/run/containerd/containerd.sock"
	binaryLogsShim = "/bin/shim-logs"
)

const (
	defaultMemory = 256 * 1024 * 1204 // 256MiB
	defaultCPU    = 1

	failuresBeforeDestroy = 4
	restartDelay          = 2 * time.Second
)

var (
	ignoreMntTypes = map[string]struct{}{
		"proc":   {},
		"tmpfs":  {},
		"devpts": {},
		"mqueue": {},
		"sysfs":  {},
	}

	// a marker value we use in failure cache to let the watcher
	// no that we don't want to try restarting this container
	permanent = struct{}{}
)

var (
	// ErrEmptyRootFS is returned when RootFS field is empty when trying to create a container
	ErrEmptyRootFS = errors.New("RootFS of the container creation data cannot be empty")

	_ pkg.ContainerModule = (*Module)(nil)
)

// Module implements pkg.Module interface
type Module struct {
	containerd string
	root       string
	client     zbus.Client
	failures   *cache.Cache
}

// New return an new pkg.ContainerModule
func New(client zbus.Client, root string, containerd string) *Module {
	if len(containerd) == 0 {
		containerd = containerdSock
	}

	module := &Module{
		containerd: containerd,
		root:       root,
		client:     client,
		// values are cached only for 1 minute. purge cache every 20 second
		failures: cache.New(time.Minute, 20*time.Second),
	}

	if err := module.upgrade(); err != nil {
		log.Error().Err(err).Msg("failed to update containers configurations")
	}

	return module
}

// upgrade containers configurations. we make sure that any configuration changes apply
// to the running containers..
func (c *Module) upgrade() error {
	// We make sure that ALL containers have auto-restart disabled since this is now
	// managed completely by the watcher.
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}

	defer client.Close()

	ctx := context.Background()
	nss, err := client.NamespaceService().List(ctx)
	if err != nil {
		return err
	}

	for _, ns := range nss {
		ctx := namespaces.WithNamespace(context.Background(), ns)
		containers, err := client.Containers(ctx)
		if err != nil {
			log.Error().Err(err).Str("namespace", ns).Msg("failed to list containers")
			continue
		}

		for _, container := range containers {
			if err := container.Update(ctx, restart.WithNoRestarts); err != nil { // mark this container as perminant down. so the watcher
				log.Warn().Err(err).Msg("failed to clear up restart task status, continuing anyways") // does not try to restart it again
			}
		}
	}

	return nil
}

// Run creates and starts a container
func (c *Module) Run(ns string, data pkg.Container) (id pkg.ContainerID, err error) {
	log.Info().Str("ns", ns).Msg("starting container")

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
		return id, errors.Wrap(err, "error updating environment variable from startup file")
	}

	opts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/amd64"),
		oci.WithRootFSPath(data.RootFS),
		oci.WithEnv(data.Env),
		oci.WithHostResolvconf,
		removeRunMount(),
		withNetworkNamespace(data.Network.Namespace),
		withMounts(data.Mounts),
		WithMemoryLimit(uint64(data.Memory)),
		WithCPUCount(data.CPU),
	}
	if data.RootFsPropagation != "" {
		opts = append(opts, WithRootfsPropagation(data.RootFsPropagation))
	}
	if data.Elevated {
		log.Warn().Msg("elevated container requested")

		opts = append(
			opts,
			oci.WithAddedCapabilities([]string{"CAP_SYS_ADMIN"}),
			oci.WithLinuxDevice("/dev/fuse", "rwm"),
		)
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
		restart.WithBinaryLogURI(binaryLogsShim, nil),
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
			_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
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

	// set user defined endpoint stats
	for _, l := range data.Stats {
		switch l.Type {
		case stats.RedisType:
			s, err := stats.NewRedis(l.Endpoint)

			if err != nil {
				log.Error().Err(err).Msg("redis stats")
				continue
			}

			go func() {
				err := stats.Monitor(c.containerd, ns, data.Name, s)
				log.Error().Err(err).Str("name", data.Name).Msg("container monitor exited with error")
			}()

		default:
			log.Error().Str("type", l.Type).Msg("invalid stats type requested")
		}
	}

	if err := c.ensureTask(ctx, container); err != nil {
		return id, err
	}

	return pkg.ContainerID(container.ID()), nil
}

// Exec executes a command inside the container
func (c *Module) Exec(ns string, containerID string, timeout time.Duration, args ...string) error {
	var timedout bool
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}
	defer client.Close()
	ctx := namespaces.WithNamespace(context.Background(), ns)
	createctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	container, err := client.LoadContainer(createctx, containerID)
	if err != nil {
		return errors.Wrap(err, "couldn't load container")
	}
	t, err := container.Task(createctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create task")
	}
	var p specs.Process
	p.Cwd = "/"
	p.Args = args
	taskID, err := uuid.NewUUID()
	if err != nil {
		return errors.Wrap(err, "failed to generate a uuid")
	}
	pr, err := t.Exec(createctx, taskID.String(), &p, cio.NullIO)
	if err != nil {
		return errors.Wrap(err, "failed to exec new porcess")
	}
	if err := pr.Start(createctx); err != nil {
		return errors.Wrap(err, "failed to start process")
	}
	ch, err := pr.Wait(createctx)
	if err != nil {
		return errors.Wrap(err, "failed to wait for process")
	}
	select {
	case <-ch:
	case <-createctx.Done():
	}
	deleteCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// if still running => execution timedout
	st, err := pr.Status(deleteCtx)
	if err != nil {
		return errors.Wrap(err, "couldn't check task state")
	}
	if st.Status != containerd.Stopped {
		timedout = true
	}
	ex, err := pr.Delete(deleteCtx, containerd.WithProcessKill)
	if err != nil {
		return errors.Wrap(err, "error deleting the created task")
	}
	if timedout {
		return errors.New("execution timed out")
	} else if ex.ExitCode() != 0 {
		return fmt.Errorf("non-zero exit code: %d", ex.ExitCode())
	}
	return nil
}
func (c *Module) ensureTask(ctx context.Context, container containerd.Container) error {
	uri, err := url.Parse("binary://" + binaryLogsShim)
	if err != nil {
		return err
	}

	log.Info().Str("loguri", uri.String()).Msg("external logging process")
	task, err := container.Task(ctx, nil)

	if err != nil && !errdefs.IsNotFound(err) {
		return err
	} else if err == nil {
		//task found, we have to stop that first
		_, err := task.Delete(ctx)
		if err != nil {
			return err
		}
	}

	//and finally create a new task
	task, err = container.NewTask(ctx, cio.LogURI(uri))
	if err != nil {
		return err
	}

	return task.Start(ctx)
}

func (c *Module) start(ns, id string) error {
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

	return c.ensureTask(ctx, container)
}

// Inspect returns the detail about a running container
func (c *Module) Inspect(ns string, id pkg.ContainerID) (result pkg.Container, err error) {
	log.Info().Str("id", string(id)).Str("ns", ns).Msg("inspect container")

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
	info, err := container.Info(ctx)
	if err != nil {
		return pkg.Container{}, err
	}

	result.CreatedAt = info.CreatedAt
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

	log.Debug().Str("id", string(id)).Str("ns", ns).Msg("container inspected")
	return
}

// ListNS list the name of all the container namespaces
func (c *Module) ListNS() ([]string, error) {
	log.Info().Msg("list namespaces")

	client, err := containerd.New(c.containerd)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx := context.Background()
	return client.NamespaceService().List(ctx)
}

// List all the existing container IDs from a certain namespace ns
func (c *Module) List(ns string) ([]pkg.ContainerID, error) {
	log.Info().Str("ns", ns).Msg("list containers")

	client, err := containerd.New(c.containerd)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	containers, err := client.Containers(ctx, "")
	if err != nil {
		return nil, err
	}

	ids := make([]pkg.ContainerID, 0, len(containers))

	for _, c := range containers {
		ids = append(ids, pkg.ContainerID(c.ID()))
	}

	log.Debug().Str("ns", ns).Msg("containers list generated")
	return ids, nil
}

// Delete stops and remove a container
func (c *Module) Delete(ns string, id pkg.ContainerID) error {
	log.Info().Str("id", string(id)).Str("ns", ns).Msg("delete container")

	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}
	defer client.Close()

	// log.Debug().Msg("fetching namespace")
	ctx := namespaces.WithNamespace(context.Background(), ns)

	// log.Debug().Str("id", string(id)).Msg("fetching container")
	container, err := client.LoadContainer(ctx, string(id))
	if err != nil {
		return err
	}

	// mark this container as perminant down. so the watcher
	// does not try to restart it again
	c.failures.Set(string(id), permanent, cache.DefaultExpiration)

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

			log.Debug().Str("id", string(id)).Int("signal", int(signal)).Msg("sending signal")
			_ = task.Kill(ctx, signal)
			trials--

			select {
			case <-exitC:
				break loop
			case <-time.After(1 * time.Second):
			}

			if trials < -5 {
				msg := fmt.Errorf("cannot stop container, SIGTERM and SIGKILL ignored")
				log.Error().Err(msg).Msg("stopping container failed")
				break loop
			}
		}

		if _, err := task.Delete(ctx); err != nil {
			return err
		}
	}

	return container.Delete(ctx)
}

func (c *Module) ensureNamespace(ctx context.Context, client *containerd.Client, namespace string) error {
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
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to open .startup.toml file")
	}
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

	data.Env = mergeEnvs(data.Env, entry.Envs())
	if data.Entrypoint == "" && entry.Entrypoint() != "" {
		data.Entrypoint = entry.Entrypoint()
	}
	if data.WorkingDir == "" && entry.WorkingDir() != "" {
		data.WorkingDir = entry.WorkingDir()
	}

	return nil
}
