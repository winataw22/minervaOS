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
)

const (
	// socketDir where vm firecracker sockets are kept
	socketDir = "/var/run/cloud-hypervisor"
	configDir = "config"
	logsDir   = "logs"

	defaultKernelArgs = "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"
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
	for i, disk := range vm.Disks {
		id := fmt.Sprintf("%d", i+2)

		drives = append(drives, Disk{
			ID:         id,
			ReadOnly:   disk.ReadOnly,
			RootDevice: disk.Root,
			Path:       disk.Path,
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
	_, err := find(id)
	return err == nil
}

func (m *Module) makeNetwork(vm *pkg.VM) ([]Interface, string, error) {
	// assume there is always at least 1 iface present

	// we do 2 things here:
	// - create the correct fc structure
	// - create the cmd line params
	//
	// for FC vms there are 2 different methods. The original one used a built-in
	// NFS module to allow setting a static ipv4 from the command line. The newer
	// method uses a custom script inside the image to set proper IP. The config
	// is also passed through the command line.

	nics := make([]Interface, 0, len(vm.Network.Ifaces))
	for i, ifcfg := range vm.Network.Ifaces {
		nics = append(nics, Interface{
			ID:  fmt.Sprintf("eth%d", i),
			Tap: ifcfg.Tap,
			Mac: ifcfg.MAC,
		})
	}

	cmdLineSections := make([]string, 0, len(vm.Network.Ifaces)+1)
	for i, ifcfg := range vm.Network.Ifaces {
		cmdLineSections = append(cmdLineSections, m.makeNetCmdLine(i, ifcfg))
	}
	dnsSection := make([]string, 0, len(vm.Network.Nameservers))
	for _, ns := range vm.Network.Nameservers {
		dnsSection = append(dnsSection, ns.String())
	}
	cmdLineSections = append(cmdLineSections, fmt.Sprintf("net_dns=%s", strings.Join(dnsSection, ",")))

	cmdline := strings.Join(cmdLineSections, " ")

	return nics, cmdline, nil
}

func (m *Module) makeNetCmdLine(idx int, ifcfg pkg.VMIface) string {
	// net_%ifacename=%ip4_cidr,$ip4_gw[,$ip4_route],$ipv6_cidr,$ipv6_gw,public|priv
	ip4Elems := make([]string, 0, 3)
	ip4Elems = append(ip4Elems, ifcfg.IP4AddressCIDR.String())
	ip4Elems = append(ip4Elems, ifcfg.IP4GatewayIP.String())
	if len(ifcfg.IP4Net.IP) > 0 {
		ip4Elems = append(ip4Elems, ifcfg.IP4Net.String())
	}

	ip6Elems := make([]string, 0, 3)
	if ifcfg.IP6AddressCIDR.IP.To16() != nil {
		ip6Elems = append(ip6Elems, ifcfg.IP6AddressCIDR.String())
		ip6Elems = append(ip6Elems, ifcfg.IP6GatewayIP.String())
	} else {
		ip6Elems = append(ip6Elems, "slaac")
	}

	privPub := "priv"
	if ifcfg.Public {
		privPub = "public"
	}

	return fmt.Sprintf("net_eth%d=%s,%s,%s", idx, strings.Join(ip4Elems, ","), strings.Join(ip6Elems, ","), privPub)
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
	machines, err := findAll()
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

	var kargs strings.Builder
	kargs.WriteString(vm.KernelArgs)
	if kargs.Len() == 0 {
		kargs.WriteString(defaultKernelArgs)
	}

	nics, args, err := m.makeNetwork(&vm)
	if err != nil {
		return err
	}

	if kargs.Len() != 0 {
		kargs.WriteRune(' ')
	}

	kargs.WriteString(args)

	machine := Machine{
		ID: vm.Name,
		Boot: Boot{
			Kernel: vm.KernelImage,
			Initrd: vm.InitrdImage,
			Args:   kargs.String(),
		},
		Config: Config{
			CPU:       CPU(vm.CPU),
			Mem:       MemMib(vm.Memory / gridtypes.Megabyte),
			HTEnabled: false,
		},
		Interfaces:  nics,
		Disks:       devices,
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

	pid, err := find(name)
	if err != nil {
		return errors.Wrapf(err, "failed to find vm with id '%s'", name)
	}

	if err := ioutil.WriteFile(filepath.Join("/proc/", fmt.Sprint(pid), "oom_adj"), []byte("-17"), 0644); err != nil {
		return errors.Wrapf(err, "failed to update oom priority for machine '%s' (PID: %d)", name, pid)
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
	pid, err := find(name)
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
			syscall.Kill(pid, syscall.SIGTERM)
		}

		if time.Since(now) > killAfter {
			log.Debug().Str("name", name).Msg("shutting vm down [sigkill]")
			syscall.Kill(pid, syscall.SIGKILL)
			break
		}

		<-time.After(1 * time.Second)
	}

	return nil
}
