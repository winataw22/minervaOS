package vm

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/vm/cloudinit"
)

const (
	// socketDir where vm firecracker sockets are kept
	socketDir = "/var/run/cloud-hypervisor"

	// ConfigDir [deprecated] is config directory name
	ConfigDir = "config"
	// logsDir is logs directory name
	logsDir = "logs"

	// cloud-init directory
	cloudInitDir = "cloud-init"
)

var (
	//defaultKernelArgs if no args are set
	defaultKernelArgs = pkg.KernelArgs{
		"rw":      "",
		"console": "ttyS0",
		"reboot":  "k",
		"panic":   "1",
		"root":    "/dev/vda",
	}
)

// Module implements the VMModule interface
type Module struct {
	root     string
	cfg      string
	client   zbus.Client
	lock     sync.Mutex
	failures *cache.Cache

	legacyMonitor LegacyMonitor
}

var (
	_ pkg.VMModule = (*Module)(nil)
)

// NewVMModule creates a new instance of vm manager
func NewVMModule(cl zbus.Client, root, config string) (*Module, error) {
	for _, dir := range []string{
		socketDir,
		filepath.Join(root, logsDir),
		filepath.Join(root, cloudInitDir),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	mod := &Module{
		root:   root,
		cfg:    config,
		client: cl,
		// values are cached only for 1 minute. purge cache every 20 second
		failures: cache.New(2*time.Minute, 20*time.Second),

		legacyMonitor: LegacyMonitor{root},
	}

	// run legacy monitor
	go mod.legacyMonitor.Monitor(context.Background())

	return mod, nil
}

func (m *Module) makeDevices(vm *pkg.VM) ([]Disk, error) {
	var drives []Disk
	if vm.Boot.Type == pkg.BootDisk {
		drives = append(drives, Disk{
			ID:         "1",
			Path:       vm.Boot.Path,
			RootDevice: true,
			ReadOnly:   false,
		})
	}
	for _, disk := range vm.Disks {
		id := fmt.Sprintf("%d", len(drives)+1)

		drives = append(drives, Disk{
			ID:       id,
			ReadOnly: false,
			Path:     disk.Path,
		})
	}

	return drives, nil
}

func (m *Module) makeVirtioFilesystems(vm *pkg.VM) ([]VirtioFS, error) {
	var result []VirtioFS
	for _, shared := range vm.Shared {

		result = append(result, VirtioFS{
			Tag:  shared.ID,
			Path: shared.Path,
		})
	}

	return result, nil
}

func (m *Module) socketPath(name string) string {
	return filepath.Join(socketDir, name)
}

func (m *Module) configPath(name string) string {
	return filepath.Join(m.cfg, name)
}

func (m *Module) logsPath(name string) string {
	return filepath.Join(m.root, logsDir, name)
}

func (m *Module) cloudInitImage(name string) string {
	return filepath.Join(m.root, cloudInitDir, name)
}

// Exists checks if firecracker process running for this machine
func (m *Module) Exists(id string) bool {
	_, err := Find(id)
	return err == nil
}

func (m *Module) makeNetwork(vm *pkg.VM, cfg *cloudinit.Configuration) ([]Interface, error) {
	// assume there is always at least 1 iface present

	// we do 2 things here:
	// - create the correct fc structure
	// - create the cmd line params
	//
	// for FC vms there are 2 different methods. The original one used a built-in
	// NFS module to allow setting a static ipv4 from the command line. The newer
	// method uses a custom script inside the image to set proper IP. The config
	// is also passed through the command line.

	v4Routes := make(map[string]string)
	v6Routes := make(map[string]string)

	hasPub := false
	for _, ifcfg := range vm.Network.Ifaces {
		hasPub = ifcfg.Public || hasPub
	}

	nics := make([]Interface, 0, len(vm.Network.Ifaces))
	for i, ifcfg := range vm.Network.Ifaces {
		nic := Interface{
			ID:  fmt.Sprintf("eth%d", i),
			Tap: ifcfg.Tap,
			Mac: ifcfg.MAC,
		}
		nics = append(nics, nic)

		cinet := cloudinit.Ethernet{
			Name:  nic.ID,
			Mac:   cloudinit.MacMatch(nic.Mac),
			DHCP4: false,
		}
		// cfg.Network = append(cfg.Network,)
		for _, ip := range ifcfg.IPs {
			cinet.Addresses = append(cinet.Addresses, ip.String())
		}

		if ifcfg.Public == hasPub {
			if ifcfg.IP4DefaultGateway != nil {
				cinet.Gateway4 = ifcfg.IP4DefaultGateway.String()
			}
		}

		if ifcfg.IP6DefaultGateway != nil {
			cinet.Gateway6 = ifcfg.IP6DefaultGateway.String()
		}

		// inserting extra routes in right places
		for _, route := range ifcfg.Routes {
			cinet.Routes = append(cinet.Routes, cloudinit.Route{
				To:  route.Net.String(),
				Via: route.Gateway.String(),
			})

			table := v4Routes
			if route.Net.IP.To4() == nil {
				table = v6Routes
			}
			gw := nic.ID
			if route.Gateway != nil {
				gw = route.Gateway.String()
			}
			table[route.Net.String()] = gw
		}

		cfg.Network = append(cfg.Network, cinet)
	}

	dnsSection := make([]string, 0, len(vm.Network.Nameservers))

	for _, ns := range vm.Network.Nameservers {
		dnsSection = append(dnsSection, ns.String())
	}

	if len(cfg.Network) > 0 {
		cfg.Network[0].Nameservers = &cloudinit.Nameservers{
			Addresses: dnsSection,
		}
	}

	return nics, nil
}

func (m *Module) tail(path string) (string, error) {
	// fetch 2k of bytes from the path ?
	// TODO: implement a better tail algo.

	const (
		tail = 2 * 1024 // 2K
	)

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "no logs available", nil
	} else if err != nil {
		return "", errors.Wrapf(err, "failed to tail file: %s", path)
	}

	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "fail to stat %s", f.Name())
	}
	offset := info.Size()
	if offset > tail {
		offset = tail
	}

	_, err = f.Seek(-offset, 2)
	if err != nil {
		return "", errors.Wrapf(err, "failed to seek file: %s", path)
	}

	logs, err := ioutil.ReadAll(f)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read logs from: %s", path)
	}

	return string(logs), nil
}

func (m *Module) withLogs(path string, err error) error {
	if err == nil {
		return nil
	}

	logs, tailErr := m.tail(path)
	if tailErr != nil {
		return errors.Wrapf(err, "failed to tail machine logs: %s", tailErr)
	}

	return errors.Wrapf(err, string(logs))
}

// List all running vms names
func (m *Module) List() ([]string, error) {
	machines, err := FindAll()
	if err != nil {
		return nil, err
	}

	legacy, err := findAllFC()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(machines)+len(legacy))

	for name := range machines {
		names = append(names, name)
	}

	for name := range legacy {
		names = append(names, name)
	}

	return names, nil
}

// Run vm
func (m *Module) Run(vm pkg.VM) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := vm.Validate(); err != nil {
		return errors.Wrap(err, "machine configuration validation failed")
	}

	ctx := context.Background()

	if m.Exists(vm.Name) {
		return fmt.Errorf("a vm with same name already exists")
	}

	cfg := cloudinit.Configuration{
		Metadata: cloudinit.Metadata{
			InstanceID: vm.Name,
			Hostname:   vm.Hostname,
		},
		Extension: cloudinit.Extension{
			Entrypoint:  vm.Entrypoint,
			Environment: vm.Environment,
		},
	}

	// TODO: user config should be added as another property on the
	// VM workload data. For now we always use the SSH_KEY from the env
	// if provided
	if key, ok := vm.Environment["SSH_KEY"]; ok {
		cfg.Users = append(cfg.Users, cloudinit.User{
			Name: "root",
			Keys: func() []string {
				// in case ssh_key container multiple keys. is this a usecase (?)
				// to be backward compatible, we split over "newlines"
				lines := strings.Split(key, "\n")
				var keys []string
				// but we also support using `,` as a separator
				for _, line := range lines {
					keys = append(keys, strings.Split(line, ",")...)
				}
				return keys
			}(),
		})
	}

	devices, err := m.makeDevices(&vm)
	if err != nil {
		return err
	}
	fs, err := m.makeVirtioFilesystems(&vm)
	if err != nil {
		return errors.Wrap(err, "failed while constructing qsfs filesystems")
	}

	cmdline := vm.KernelArgs
	if cmdline == nil {
		cmdline = pkg.KernelArgs{}
		cmdline.Extend(defaultKernelArgs)
	}
	var env map[string]string
	if vm.Boot.Type == pkg.BootVirtioFS {
		// booting from a virtiofs. the vm is basically
		// running as a container. hence we set extra cmdline
		// arguments
		cmdline["root"] = virtioRootFsTag
		cmdline["rootfstype"] = "virtiofs"

		// we add the fs for booting.
		fs = append(fs, VirtioFS{
			Tag:  virtioRootFsTag,
			Path: vm.Boot.Path,
		})
		// we set the environment
		env = vm.Environment
		// add we also add disk mounts
		for i, mnt := range vm.Disks {
			name := fmt.Sprintf("/dev/vd%c", 'a'+i)
			cfg.Mounts = append(cfg.Mounts,
				cloudinit.Mount{
					Source: name,
					Target: mnt.Target,
					Type:   cloudinit.MountTypeAuto,
				})
		}
		for _, q := range vm.Shared {
			cfg.Mounts = append(cfg.Mounts,
				cloudinit.Mount{
					Source: q.ID,
					Target: q.Target,
					Type:   cloudinit.MountTypeVirtiofs,
				})
		}
	}

	nics, err := m.makeNetwork(&vm, &cfg)
	if err != nil {
		return err
	}

	ciImage := m.cloudInitImage(vm.Name)

	if err := cloudinit.CreateImage(ciImage, cfg); err != nil {
		return errors.Wrap(err, "failed to create cloud-init image")
	}

	devices = append(devices, Disk{
		ID:       fmt.Sprintf("%d", len(devices)+1),
		Path:     ciImage,
		ReadOnly: true,
	})

	machine := Machine{
		ID: vm.Name,
		Boot: Boot{
			Kernel: vm.KernelImage,
			Initrd: vm.InitrdImage,
			Args:   cmdline.String(),
		},
		Config: Config{
			CPU:       CPU(vm.CPU),
			Mem:       MemMib(vm.Memory / gridtypes.Megabyte),
			HTEnabled: false,
		},
		Entrypoint:  vm.Entrypoint,
		FS:          fs,
		Interfaces:  nics,
		Disks:       devices,
		Environment: env,
		NoKeepAlive: vm.NoKeepAlive,
	}

	log.Debug().Str("name", vm.Name).Msg("saving machine")
	if err := machine.Save(m.configPath(vm.Name)); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			log.Error().Err(err).Msg("decomission duo to failure to create the VM")
			_ = m.Delete(machine.ID)
		}
	}()

	if vm.NoKeepAlive {
		m.failures.Set(vm.Name, permanent, cache.NoExpiration)
	}

	if err = machine.Run(ctx, m.socketPath(vm.Name), m.logsPath(vm.Name)); err != nil {
		return m.withLogs(m.logsPath(vm.Name), err)
	}

	if err := m.waitAndAdjOom(ctx, vm.Name); err != nil {
		return m.withLogs(m.logsPath(vm.Name), err)
	}

	return nil
}

func (m *Module) waitAndAdjOom(ctx context.Context, name string) error {
	check := func() error {
		if !m.Exists(name) {
			return fmt.Errorf("failed to spawn vm machine process '%s'", name)
		}
		//TODO: check unix connection
		socket := m.socketPath(name)
		con, err := net.Dial("unix", socket)
		if err != nil {
			return err
		}

		con.Close()
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	if err := backoff.Retry(check, backoff.WithContext(backoff.NewConstantBackOff(2*time.Second), ctx)); err != nil {
		return err
	}

	ps, err := Find(name)
	if err != nil {
		return errors.Wrapf(err, "failed to find vm with id '%s'", name)
	}

	if err := ioutil.WriteFile(filepath.Join("/proc/", fmt.Sprint(ps.Pid), "oom_adj"), []byte("-17"), 0644); err != nil {
		return errors.Wrapf(err, "failed to update oom priority for machine '%s' (PID: %d)", name, ps.Pid)
	}

	return nil
}

// Logs returns machine logs for give machine name
func (m *Module) Logs(name string) (string, error) {
	path := m.logsPath(name)
	return m.tail(path)
}

// Inspect a machine by name
func (m *Module) Inspect(name string) (pkg.VMInfo, error) {
	if !m.Exists(name) {
		return pkg.VMInfo{}, fmt.Errorf("machine '%s' does not exist", name)
	}
	client := NewClient(m.socketPath(name))
	cpu, mem, err := client.Inspect(context.Background())
	if err != nil {
		return pkg.VMInfo{}, errors.Wrap(err, "failed to get machine configuration")
	}

	return pkg.VMInfo{
		CPU:       int64(cpu),
		Memory:    int64(mem),
		HtEnabled: false,
	}, nil
}

func (m *Module) removeConfig(name string) {
	if name == "" {
		return
	}

	_ = os.Remove(m.configPath(name))

	_ = os.Remove(m.cloudInitImage(name))

	_ = os.Remove(m.logsPath(name))
}

// Delete deletes a machine by name (id)
func (m *Module) Delete(name string) error {
	defer m.failures.Delete(name)

	// before we do anything we set failures to permanent to prevent monitoring from trying
	// to revive this machine
	m.failures.Set(name, permanent, cache.NoExpiration)
	defer m.removeConfig(name)

	//is this the real life? is this just legacy?
	if pid, err := findFC(name); err == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return m.legacyMonitor.cleanFsFirecracker(name)
	}

	// normal operation
	ps, err := Find(name)
	if err != nil {
		// machine already gone
		return nil
	}

	client := NewClient(m.socketPath(name))

	// timeout is request timeout, not machine timeout to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	now := time.Now()

	const (
		termAfter = 5 * time.Second
		killAfter = 10 * time.Second
	)

	log.Debug().Str("name", name).Msg("shutting vm down [client]")
	if err := client.Shutdown(ctx); err != nil {
		log.Error().Err(err).Str("name", name).Msg("failed to shutdown machine")
	}

	for {
		if !m.Exists(name) {
			return nil
		}

		log.Debug().Str("name", name).Msg("shutting vm down [sigterm]")
		if time.Since(now) > termAfter {
			_ = syscall.Kill(ps.Pid, syscall.SIGTERM)
		}

		if time.Since(now) > killAfter {
			log.Debug().Str("name", name).Msg("shutting vm down [sigkill]")
			_ = syscall.Kill(ps.Pid, syscall.SIGKILL)
			break
		}

		<-time.After(1 * time.Second)
	}

	return nil
}
