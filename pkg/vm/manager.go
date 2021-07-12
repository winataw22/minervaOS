package vm

import (
	"bytes"
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
)

const (
	// socketDir where vm firecracker sockets are kept
	socketDir = "/var/run/cloud-hypervisor"
	configDir = "config"
	logsDir   = "logs"
)

var (
	//defaultKernelArgs if no args are set
	defaultKernelArgs = pkg.KernelArgs{
		"rw":      "",
		"console": "ttyS0",
		"reboot":  "k",
		"panic":   "1",
	}
)
var (
	protectedKernelEnv = map[string]struct{}{
		"init":       {},
		"root":       {},
		"rootfstype": {},
		"console":    {},
		"net_eth1":   {},
		"net_eth2":   {},
		"net_dns":    {},
		"panic":      {},
		"reboot":     {},
	}
)

// Module implements the VMModule interface
type Module struct {
	root     string
	client   zbus.Client
	lock     sync.Mutex
	failures *cache.Cache

	legacyMonitor LegacyMonitor
}

var (
	_ pkg.VMModule = (*Module)(nil)
)

// NewVMModule creates a new instance of vm manager
func NewVMModule(cl zbus.Client, root string) (*Module, error) {
	for _, dir := range []string{
		socketDir,
		filepath.Join(root, configDir),
		filepath.Join(root, logsDir),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	mod := &Module{
		root:   root,
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

func (m *Module) socketPath(name string) string {
	return filepath.Join(socketDir, name)
}

func (m *Module) configPath(name string) string {
	return filepath.Join(m.root, configDir, name)
}

func (m *Module) logsPath(name string) string {
	return filepath.Join(m.root, logsDir, name)
}

// Exists checks if firecracker process running for this machine
func (m *Module) Exists(id string) bool {
	_, err := Find(id)
	return err == nil
}

func (m *Module) buildRouteParam(defaultGw net.IP, table map[string]string) string {
	var buf bytes.Buffer
	if defaultGw != nil {
		buf.WriteString(fmt.Sprintf("default,%s", defaultGw.String()))
	}

	for k, v := range table {
		if buf.Len() > 0 {
			buf.WriteRune(';')
		}
		buf.WriteString(k)
		buf.WriteRune(',')
		buf.WriteString(v)
	}

	return buf.String()
}

func (m *Module) makeNetwork(vm *pkg.VM) ([]Interface, pkg.KernelArgs, error) {
	// assume there is always at least 1 iface present

	// we do 2 things here:
	// - create the correct fc structure
	// - create the cmd line params
	//
	// for FC vms there are 2 different methods. The original one used a built-in
	// NFS module to allow setting a static ipv4 from the command line. The newer
	// method uses a custom script inside the image to set proper IP. The config
	// is also passed through the command line.

	args := pkg.KernelArgs{}
	v4Routes := make(map[string]string)
	v6Routes := make(map[string]string)
	var defaultGw4 net.IP
	var defaultGw6 net.IP

	nics := make([]Interface, 0, len(vm.Network.Ifaces))
	for i, ifcfg := range vm.Network.Ifaces {
		nic := Interface{
			ID:  fmt.Sprintf("eth%d", i),
			Tap: ifcfg.Tap,
			Mac: ifcfg.MAC,
		}
		nics = append(nics, nic)

		var ips []string
		for _, ip := range ifcfg.IPs {
			ips = append(ips, ip.String())
		}
		// configure nic ips
		args[fmt.Sprintf("net_%s", nic.ID)] = strings.Join(ips, ";")
		// configure nic routes
		if defaultGw4 == nil && ifcfg.IP4DefaultGateway != nil {
			defaultGw4 = ifcfg.IP4DefaultGateway
		}

		if defaultGw6 == nil && ifcfg.IP6DefaultGateway != nil {
			defaultGw6 = ifcfg.IP6DefaultGateway
		}
		// one extra check to always use public nic as default
		// gw
		if ifcfg.Public && ifcfg.IP4DefaultGateway != nil {
			defaultGw4 = ifcfg.IP4DefaultGateway
		}

		// inserting extra routes in right places
		for _, route := range ifcfg.Routes {
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
	}

	args["net_r4"] = m.buildRouteParam(defaultGw4, v4Routes)
	args["net_r6"] = m.buildRouteParam(defaultGw6, v6Routes)

	dnsSection := make([]string, 0, len(vm.Network.Nameservers))
	for _, ns := range vm.Network.Nameservers {
		dnsSection = append(dnsSection, ns.String())
	}
	args["net_dns"] = strings.Join(dnsSection, ";")

	return nics, args, nil
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

	devices, err := m.makeDevices(&vm)
	if err != nil {
		return err
	}

	cmdline := vm.KernelArgs
	if cmdline == nil {
		cmdline = pkg.KernelArgs{}
		cmdline.Extend(defaultKernelArgs)
	}

	var fs []VirtioFS
	var env map[string]string
	if vm.Boot.Type == pkg.BootVirtioFS {
		// booting from a virtiofs. the vm is basically
		// running as a container. hence we set extra cmdline
		// arguments
		cmdline["root"] = virtioRootFsTag
		cmdline["rootfstype"] = "virtiofs"

		// we add the fs for booting.
		fs = []VirtioFS{
			{Tag: virtioRootFsTag, Path: vm.Boot.Path},
		}
		// we set the environment
		env = vm.Environment
		// add we also add disk mounts
		for i, mnt := range vm.Disks {
			name := fmt.Sprintf("vd%c", 'a'+i)
			cmdline[name] = mnt.Target
		}
	} else {
		// if with no virtio fs we can only
		// set the given environment to the linux kernel
		// but this is not safe.
		// TODO: Should we only allow UPPER_CASE
		// env to pass to avoid overriding other params ?!
		for k, v := range vm.Environment {
			if strings.HasPrefix(k, "vd") {
				continue
			}
			if _, ok := protectedKernelEnv[k]; ok {
				continue
			}
			cmdline[k] = v
		}
	}

	nics, args, err := m.makeNetwork(&vm)
	if err != nil {
		return err
	}

	cmdline.Extend(args)

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
			m.Delete(machine.ID)
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

// Delete deletes a machine by name (id)
func (m *Module) Delete(name string) error {
	defer m.failures.Delete(name)

	// before we do anything we set failures to permanent to prevent monitoring from trying
	// to revive this machine
	m.failures.Set(name, permanent, cache.NoExpiration)
	defer os.RemoveAll(m.configPath(name))

	//is this the real life? is this just legacy?
	if pid, err := findFC(name); err == nil {
		syscall.Kill(pid, syscall.SIGKILL)
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
			syscall.Kill(ps.Pid, syscall.SIGTERM)
		}

		if time.Since(now) > killAfter {
			log.Debug().Str("name", name).Msg("shutting vm down [sigkill]")
			syscall.Kill(ps.Pid, syscall.SIGKILL)
			break
		}

		<-time.After(1 * time.Second)
	}

	return nil
}
